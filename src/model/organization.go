package model

import (
	"database/sql/driver"
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
	"mictract/config"
	"mictract/global"
	"mictract/model/kubernetes"
	"path/filepath"
	"strings"
	"time"
)

type Organization struct {
	ID int `json:"id"`
	NetworkID	int	`json:"networkid"`
	Name        string `json:"name"`
	Nickname	string `json:"nickname"`
	MSPID       string `json:"mspid"`

	Peers       Peers  `json:"peers"`
	Users		[]string `json:"users"`

	Status	string `json:"status"`

	//CAID        string `json:"caid"`
	//NetworkName string `json:"networkname"`
}

type Organizations []Organization

// 自定义数据字段所需实现的两个接口
func (orgs *Organizations) Scan(value interface{}) error {
	return scan(&orgs, value)
}
func (orgs Organizations) Value() (driver.Value, error) {
	return value(orgs)
}
func (org *Organization) Scan(value interface{}) error {
	return scan(&org, value)
}
func (org Organization) Value() (driver.Value, error) {
	return value(org)
}

func (org *Organization) GetMSPPath() string {
	ret := filepath.Join(config.LOCAL_BASE_PATH, fmt.Sprintf("net%d", org.NetworkID))
	if org.ID == -1 {
		ret = filepath.Join(ret, "ordererOrganizations", fmt.Sprintf("net%d.com", org.NetworkID))
	} else {
		ret = filepath.Join(ret, "peerOrganizations", fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID))
	}
	return filepath.Join(ret, "msp")
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

// eg: GetBasicOrg(1, 1).CreateBasicOrganizationEntity()
// eg: GetBasicOrg(1, 1).CreateNodeEntity()
func GetBasicOrg(orgID, netID int, nickname string) *Organization {
	if orgID == -1 {
		return &Organization{
			ID: orgID,
			NetworkID: netID,
			Name: "ordererorg",
			Nickname: nickname,
			MSPID: "ordererMSP",
			Peers: []Peer{},
			Users: []string{},
			Status: "starting",
		}
	} else {
		return &Organization{
			ID: orgID,
			Name: fmt.Sprintf("org%d", orgID),
			Nickname: nickname,
			NetworkID: netID,
			MSPID: fmt.Sprintf("org%dMSP", orgID),
			Peers: []Peer{},
			Users: []string{},
			Status: "starting",
		}
	}
}

// CreateBasicOrganizationEntity starts a CA node,
// and registers the node certificate
// and two user certificates, one administrator user and one ordinary user
func (org *Organization)CreateBasicOrganizationEntity() error {
	global.Logger.Info(fmt.Sprintf("Starting a new organization org%d...", org.ID))

	UpdateNets(*org)

	model := kubernetes.NewOrdererCA(org.NetworkID)
	if org.ID != -1 {
		model = kubernetes.NewPeerCA(org.NetworkID, org.ID)
	}
	//model.Create()
	global.Logger.Info("peer ca starts creating")
	if err := model.AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("peer ca has been created synchronously")

	// Wait for the ca program to start
	time.Sleep(15 * time.Second)

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
		NewAdminCaUser(1, org.ID, org.NetworkID, "admin1pw"),
		// NewUserCaUser(1, org.ID, org.NetworkID, "user1pw"),
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
	if org.ID == -1 {
		global.Logger.Info("orderer starts creating")
		if err := kubernetes.NewOrderer(org.NetworkID, 1).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("orderer has been created synchronously")
	} else {
		global.Logger.Info("peer starts creating")
		if err := kubernetes.NewPeer(org.NetworkID, org.ID, 1).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("peer has been created synchronously")
	}

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

func GetOrgFromNets(orgID, netID int) (Organization, error) {
	net, err := GetNetworkfromNets(netID)
	if err != nil {
		return Organization{}, err
	}
	if orgID <= 0 || orgID + 1 > len(net.Organizations) {
		return Organization{}, errors.New("Can't find the org")
	}
	return net.Organizations[orgID], nil
}

func (o *Organization)AddPeer() error {
	if o.ID == -1 {
		return errors.New("Just for peer, not orderer")
	}

	org, err := GetOrgFromNets(o.ID, o.NetworkID)
	if err != nil {
		return err
	}

	if err := UpdateSDK(org.NetworkID); err != nil {
		return err
	}

	sdk, err := GetSDKByNetWorkID(org.NetworkID)
	if err != nil {
		return err
	}

	orgName := fmt.Sprintf("org%d", org.ID)
	caURL := fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(caURL), mspclient.WithOrg(orgName))
	if err != nil {
		return err
	}

	newPeer := NewPeerCaUser(len(org.Peers) + 1, org.ID, org.NetworkID, "peer1pw")
	if err := newPeer.Register(mspClient); err != nil {
		return errors.WithMessage(err, "fail to regiester new Peer")
	}

	if err := newPeer.Enroll(mspClient, false); err != nil {
		return err
	}
	if err := newPeer.Enroll(mspClient, true); err != nil {
		return err
	}

	// org.Peers = append(org.Peers, Peer{Name: newPeer.GetUsername()})
	global.Logger.Info("peer starts creating")
	if err := kubernetes.NewPeer(newPeer.NetworkID, newPeer.OrganizationID, newPeer.UserID).AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("peer has been created synchronously")

	return nil
}