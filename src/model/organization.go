package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"mictract/global"
	"mictract/model/kubernetes"
	"path/filepath"
	"strings"
	"time"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"
)

type Organization struct {
	ID int `json:"id"`
	NetworkID	int	`json:"networkid"`
	Name        string `json:"name"`
	MSPID       string `json:"mspid"`

	Peers       Peers  `json:"peers"`
	Users		[]string `json:"users"`

	//CAID        string `json:"caid"`
	//NetworkName string `json:"networkname"`
}

type Organizations []Organization

// 自定义数据字段所需实现的两个接口
func (orgs *Organizations) Scan(value interface{}) error {
	return scan(&orgs, value)
}

func (orgs *Organizations) Value() (driver.Value, error) {
	return value(orgs)
}

func (org *Organization) GetMSPPath() string {
	ret := NewCaUserFromDomainName(fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID))
	if org.ID == -1 {
		ret = NewCaUserFromDomainName(fmt.Sprintf("Admin1@net%d.com", org.NetworkID))
	}
	return filepath.Join(ret.GetBasePath(), "..", "msp")
}

func (org *Organization) GetConfigtxFile() string {
	var configtxTemplate = `
Organizations:
    - &<OrgName>
        Name: <MSPID>
        ID: <MSPID>
        MSPDir: <MSPPath>
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('<MSPID>.admin', '<MSPID>.peer', '<MSPID>.client')"
            Writers:
                Type: Signature
                Rule: "OR('<MSPID>.admin', '<MSPID>.client')"
            Admins:
                Type: Signature
                Rule: "OR('<MSPID>.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('<MSPID>.peer')"
`
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<OrgName>", fmt.Sprintf("org%d", org.ID))
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPID>", org.MSPID)
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPPath>", org.GetMSPPath())
	return configtxTemplate
}

func (org *Organization) NewMspClient() (*mspclient.Client, error) {
	sdk, err := GetSDKByNetWorkID(org.NetworkID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk")
	}
	caID := ""
	if org.ID == -1 {
		caID = fmt.Sprintf("ca.net%d.com", org.NetworkID)
	} else {
		caID = fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	}
	return mspclient.New(sdk.Context(), mspclient.WithCAInstance(caID), mspclient.WithOrg(org.Name))
}

func (org *Organization) GetAdminSigningIdentity() (msp.SigningIdentity, error){
	adminUsername := fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID)
	if org.ID == -1 {
		adminUsername = fmt.Sprintf("Admin1@net%d.com", org.NetworkID)
	}
	mspClient, err := org.NewMspClient()
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get mspClient "+org.Name)
	}
	adminIdentity, err := mspClient.GetSigningIdentity(adminUsername)
	if err != nil {
		return nil, errors.WithMessage(err, org.Name+"fail to sign")
	}
	return adminIdentity, nil
}

// CreateBasicOrganizationEntity starts a CA node,
// and registers the node certificate
// and two user certificates, one administrator user and one ordinary user
func (org *Organization)CreateBasicOrganizationEntity() error {
	global.Logger.Info(fmt.Sprintf("Starting a new organization org%d...", org.ID))

	UpdateNets(*org)

	global.Logger.Info("Starting ca node...")
	global.Logger.Info("此处需要同步，如果你看到这条信心，不要忘了增加同步代码，并且删除这条info")
	model := kubernetes.NewOrdererCA(org.NetworkID)
	if org.ID != -1 {
		model = kubernetes.NewPeerCA(org.NetworkID, org.ID)
	}
	model.Create()

	// TODO: make it sync
	// wait for pulling images when first deploy
	time.Sleep(30 * time.Second)

	// 从ca的挂载目录里取出ca证书，构建组织msp
	global.Logger.Info("The msp of the organization is being built...")
	causer := CaUser{
		OrganizationID: org.ID,
		NetworkID: org.NetworkID,
	}

	//fmt.Println(causer, causer.GetOrgMspDir())
	if err := causer.GenerateOrgMsp(); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	// 更新它的sdk，方便下一步获取最新的sdk
	UpdateSDK(org.NetworkID)

	sdk, err := GetSDKByNetWorkID(org.NetworkID)
	if err != nil {
		return err
	}

	orgName := fmt.Sprintf("org%d", org.ID)
	caURL := fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	if org.ID == -1 {
		caURL = fmt.Sprintf("ca.net%d.com", org.NetworkID)
		orgName = "ordererorg"
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(caURL), mspclient.WithOrg(orgName))
	if err != nil {
		return err
	}

	// register users of this organization
	global.Logger.Info("Registering users...")
	users := []*CaUser{
		NewUserCaUser(1, org.ID, org.NetworkID, "user1pw"),
		NewAdminCaUser(1, org.ID, org.NetworkID, "admin1pw"),
		// NewPeerCaUser(1, org.ID, org.NetworkID, "peer1pw"),
	}
	if org.ID == -1 {
		users = append(users, NewOrdererCaUser(1, org.NetworkID, "orderer1pw"))
	} else {
		users = append(users, NewPeerCaUser(1, org.ID, org.NetworkID, "peer1pw"))
	}

	for _, u := range users {
		if err := u.Register(mspClient); err != nil {
			return err
		}
	}

	// enroll to build msp and tls directories
	global.Logger.Info("User certificate is being stored...")
	for _, u := range users {
		// msp
		if err := u.Enroll(mspClient, false); err != nil {
			return err
		}
		// tls
		if err := u.Enroll(mspClient, true); err != nil {
			return err
		}
	}

	return nil
}

// CreateNodeEntity creates a node entity and starts the peer or orderer node in the organization through the existing certificate
func (org *Organization) CreateNodeEntity() error {
	global.Logger.Info("Starting the peer node or orderer node")
	global.Logger.Info("此处可能需要同步，启动节点后，如果用户立即对节点进行操作，可能pod里节点程序未初始化完成，导致错误。如果你看到这条信息，不要忘了考虑这件事情，并且删除这条info")
	if org.ID == -1 {
		kubernetes.NewOrderer(org.NetworkID, 1).Create()
	} else {
		kubernetes.NewPeer(org.NetworkID, org.ID, 1).Create()
	}

	// TODO: make it sync
	// wait for pulling images when first deploy
	time.Sleep(5 * time.Second)

	return nil
}

func (org *Organization) RemoveAllEntity() {
	global.Logger.Info("Entity is being removed...")

	if org.ID != -1 {
		global.Logger.Info("Remove PeerCA...")
		kubernetes.NewPeerCA(org.NetworkID, org.ID).Delete()
		for _, peer := range org.Peers {
			global.Logger.Info("Remove " + peer.Name + "...")
			user := NewCaUserFromDomainName(peer.Name)
			kubernetes.NewPeer(user.NetworkID, user.OrganizationID, user.UserID).Delete()
		}
	} else {
		global.Logger.Info("Remove OrdererCA...")
		kubernetes.NewOrdererCA(org.NetworkID).Delete()
		for _, peer := range org.Peers {
			global.Logger.Info("Remove " + peer.Name + "...")
			user := NewCaUserFromDomainName(peer.Name)
			kubernetes.NewOrderer(user.NetworkID, user.UserID).Delete()
		}
	}
}