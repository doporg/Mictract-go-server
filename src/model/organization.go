package model

import (
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mictract/config"
	"mictract/enum"
	"mictract/global"
	"mictract/model/kubernetes"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Organization struct {
	ID 					int 	`json:"id"`
	NetworkID			int		`json:"network_id"`
	Nickname			string 	`json:"nickname"`
	Status				string 	`json:"status"`

	CreatedAt 			time.Time
	IsOrdererOrg	 	bool
}

func NewOrganization(netID int, nickname string) (*Organization, error) {
	// 1. TODO: check netID exists or not
	net, _ := FindNetworkByID(netID)
	if net.Status == enum.StatusError {
		return &Organization{}, errors.New("Failed to call NewOrganization, network status is abnormal")
	}

	// 2. new
	org := &Organization{
		NetworkID: 		netID,
		Nickname: 		nickname,
		Status: 		"starting",
		CreatedAt: 		time.Now(),
		IsOrdererOrg: 	false,
	}

	// 3. insert into db
	if err := global.DB.Create(org).Error; err != nil {
		return &Organization{}, err
	}

	return org, nil
}

func NewOrdererOrganization(netID int, nickname string) (*Organization, error) {
	// 1. TODO: check netID exists or not
	net, _ := FindNetworkByID(netID)
	if net.Status == enum.StatusError {
		return &Organization{}, errors.New("Failed to call NewOrdererOrganization, network status is abnormal")
	}

	// 2. new
	org := &Organization{
		NetworkID: 		netID,
		Nickname:	 	nickname,
		Status: 		enum.StatusStarting,
		CreatedAt: 		time.Now(),
		IsOrdererOrg:   true,
	}

	// 3. insert into db
	if err := global.DB.Create(org).Error; err != nil {
		return &Organization{}, err
	}

	return org, nil
}

func (org *Organization) NewMspClient() (*mspclient.Client, error) {
	sdk, err := GetSDKByNetworkID(org.NetworkID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk")
	}
	return mspclient.New(sdk.Context(), mspclient.WithCAInstance(org.GetCAID()), mspclient.WithOrg(org.GetName()))
}

func (org *Organization) IsOrdererOrganization() bool {
	return org.IsOrdererOrg
}

func (org *Organization) GetName() string {
	if org.IsOrdererOrganization() {
		return  fmt.Sprintf("ordererorg")
	} else {
		return fmt.Sprintf("org%d", org.ID)
	}
}

func (org *Organization) GetSystemUser() (*CaUser, error) {
	var sysUsers []CaUser
	if err := global.DB.
		Where("nickname = ? and organization_id = ?", "system-user", org.ID).
		Find(&sysUsers).Error; err != nil {
		return &CaUser{}, err
	}
	return &sysUsers[0], nil
}

func (org *Organization) GetPeers() ([]CaUser, error) {
	peers, err := FindCaUserInOrganization(org.ID, org.NetworkID, "peer")
	if err != nil {
		return []CaUser{}, err
	}
	if len(peers) <= 0 {
		return []CaUser{}, errors.New("No peer in org")
	}
	return peers, nil
}

func (org *Organization) GetOrderers() ([]CaUser, error) {
	peers, err := FindCaUserInOrganization(org.ID, org.NetworkID, "orderer")
	if err != nil {
		return []CaUser{}, err
	}
	if len(peers) <= 0 {
		return []CaUser{}, errors.New("No orderer in org")
	}
	return peers, nil
}

func (org *Organization) GetUsers() ([]CaUser, error) {
	users1, err := FindCaUserInOrganization(org.ID, org.NetworkID, "user")
	if err != nil {
		return []CaUser{}, err
	}
	users2, err := FindCaUserInOrganization(org.ID, org.NetworkID, "admin")
	if err != nil {
		return []CaUser{}, err
	}
	return append(users1, users2...), nil
}

func (org *Organization) GetMSPID() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("ordererMSP")
	} else {
		return fmt.Sprintf("org%dMSP", org.ID)
	}
}

func (org *Organization) GetCAID() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("ca.net%d.com", org.NetworkID)
	} else {
		return fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	}
}

func (org *Organization) GetCAURLInK8S() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("https://ca-net%d:7054", org.NetworkID)
	} else {
		return fmt.Sprintf("https://ca-org%d-net%d:7054", org.ID, org.NetworkID)
	}
}

func (org *Organization) GetMSPPath() string {
	ret := filepath.Join(config.LOCAL_BASE_PATH, fmt.Sprintf("net%d", org.NetworkID))
	if org.IsOrdererOrganization() {
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
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<OrgName>", org.GetName())
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPID>",   org.GetMSPID())
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPPath>", org.GetMSPPath())
	return configtxTemplate
}

func (org *Organization) GetAdminSigningIdentity() (msp.SigningIdentity, error){
	adminUsername := fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID)
	if org.ID == -1 {
		adminUsername = fmt.Sprintf("Admin1@net%d.com", org.NetworkID)
	}
	mspClient, err := org.NewMspClient()
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get mspClient "+org.GetName())
	}
	adminIdentity, err := mspClient.GetSigningIdentity(adminUsername)
	if err != nil {
		return nil, errors.WithMessage(err, org.GetName()+"fail to sign")
	}
	return adminIdentity, nil
}

func (org *Organization) GetMSPDir() string {
	netName := fmt.Sprintf("net%d", org.NetworkID)

	basePath := filepath.Join(config.LOCAL_BASE_PATH, netName)

	if org.IsOrdererOrganization() {
		// ordererOrganizations
		basePath = filepath.Join(basePath, "ordererOrganizations", netName + ".com")
	} else {
		// peerOrganizations
		basePath = filepath.Join(basePath, "peerOrganizations",
			fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID))
	}

	// Build MSP directory by the given CaUser.
	return filepath.Join(basePath, "msp")
}

func (org *Organization) GenerateOrgMSP() error {
	basePath := org.GetMSPDir()

	// cacerts/ca-cert.pem
	if err := os.MkdirAll(filepath.Join(basePath, "cacerts"), os.ModePerm); err != nil {
		return err
	}
	if _, err := copy(filepath.Join(basePath, "..", "ca", "ca-cert.pem"), filepath.Join(basePath, "cacerts", "ca-cert.pem")); err != nil {
		return err
	}

	//fmt.Println(cu, cu.GetOrgMspDir(), basePath, "..", "ca", "ca-cert.pem")

	// tlscacerts/tlsca-cert.pem
	if err := os.MkdirAll(filepath.Join(basePath, "tlscacerts"), os.ModePerm); err != nil {
		return err
	}
	if _, err := copy(filepath.Join(basePath, "..", "ca", "ca-cert.pem"), filepath.Join(basePath, "tlscacerts", "tlsca-cert.pem")); err != nil {
		return err
	}

	// config.yaml
	f3, err := os.Create(filepath.Join(basePath, "config.yaml"))
	if err != nil {
		return err
	}
	defer f3.Close()
	ouconfig := `NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: orderer`
	_, _ = f3.Write([]byte(ouconfig))

	return nil
}

func (org *Organization) GetCACert() string {
	content, err := ioutil.ReadFile(filepath.Join(org.GetMSPDir(), "cacerts", "ca-cert.pem"))
	if err != nil {
		global.Logger.Error("fail to read ca-cert.pem", zap.Error(err))
	}
	return string(content)
}

// CreateBasicOrganizationEntity starts a CA node,
// and registers the node certificate and admin certificates
func (org *Organization)CreateBasicOrganizationEntity() error {
	global.Logger.Info(fmt.Sprintf("Starting a new organization org%d...", org.ID))

	// 1. create ca pod
	model := kubernetes.NewOrdererCA(org.NetworkID)
	if !org.IsOrdererOrganization() {
		model = kubernetes.NewPeerCA(org.NetworkID, org.ID)
	}
	global.Logger.Info("peer ca starts creating")
	if err := model.AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("peer ca has been created synchronously")

	// 2. Wait for the ca program to start
	time.Sleep(15 * time.Second)

	// 3. 从ca的挂载目录里取出ca证书，构建组织msp
	global.Logger.Info("The msp of the organization is being built...")
	if err := org.GenerateOrgMSP(); err != nil {
		return err
	}

	// 4. get mspclient
	mspClient, err := org.NewMspClient()
	if err != nil {
		return err
	}

	// 5. register Admin1@org%d.net%d.com or Admin1@net%d.com
	global.Logger.Info("Registering users...")
	newUser, err := NewAdminCaUser(org.ID, org.NetworkID, "system-user", "admin1pw", org.IsOrdererOrg)
	if err != nil {
		return err
	}
	users := []*CaUser{newUser}
	if org.IsOrdererOrganization() {
		newUser, err := NewOrdererCaUser(org.ID, org.NetworkID, "orderer1pw")
		if err != nil {
			return err
		}
		users = append(users, newUser)
	} else {
		newUser, err := NewPeerCaUser(org.ID, org.NetworkID, "peer1pw")
		if err != nil {
			return err
		}
		users = append(users, newUser)
	}

	for _, u := range users {
		if err := u.Register(mspClient); err != nil {
			return err
		}
	}

	// 6. enroll to build msp and tls directories
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

// CreateNodeEntity creates a node entity and starts the peer or orderer node
// in the organization through the existing certificate
func (org *Organization) CreateNodeEntity() error {
	global.Logger.Info("Starting the peer node or orderer node")
	if org.IsOrdererOrganization() {
		global.Logger.Info("orderer starts creating")
		// 此时应该只有一个
		orderers, err := org.GetOrderers()
		if err != nil {
			return err
		}
		if err := kubernetes.NewOrderer(org.NetworkID, orderers[0].ID).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("orderer has been created synchronously")
	} else {
		global.Logger.Info("peer starts creating")
		// 此时应该只有一个
		peers, err := org.GetPeers()
		if err != nil {
			return err
		}
		if err := kubernetes.NewPeer(org.NetworkID, org.ID, peers[0].ID).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("peer has been created synchronously")
	}

	return nil
}

func (org *Organization) RemoveAllEntity() {
	if !org.IsOrdererOrganization() {
		global.Logger.Info("Remove PeerCA...")
		// 1. get users
		users, err := org.GetPeers()
		if err != nil {
			global.Logger.Error("", zap.Error(err))
		}

		// 2. remove ca entity
		kubernetes.NewPeerCA(org.NetworkID, org.ID).Delete()

		// 3. remove peer entity
		for _, peer := range users {
			global.Logger.Info("Remove " + peer.GetName() + "...")
			kubernetes.NewPeer(peer.NetworkID, peer.OrganizationID, peer.ID).Delete()
		}
	} else {
		global.Logger.Info("Remove OrdererCA...")
		// 1. get users
		net, err := FindNetworkByID(org.NetworkID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
		}
		users, err := net.GetOrderers()
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			return
		}

		// 2. remove ca entity
		kubernetes.NewOrdererCA(org.NetworkID).Delete()

		// 3. remove orderer entity
		for _, orderer := range users {
			global.Logger.Info("Remove " + orderer.GetName() + "...")
			kubernetes.NewOrderer(orderer.NetworkID, orderer.ID).Delete()
		}
	}
}

func (org *Organization)AddPeer() (*CaUser, error) {
	// 1. check
	if org.IsOrdererOrganization() {
		return &CaUser{}, errors.New("Just for peer, not orderer")
	}

	// 2. get org object
	org, err := FindOrganizationByID(org.ID)
	if err != nil {
		return &CaUser{}, err
	}

	// 3. get mspclient
	mspClient, err := org.NewMspClient()
	if err != nil {
		return &CaUser{}, err
	}

	// 4. register and enroll
	newPeer, err := NewPeerCaUser(org.ID, org.NetworkID, "peer1pw")
	if err != nil {
		return &CaUser{}, err
	}
	if err := newPeer.Register(mspClient); err != nil {
		return &CaUser{}, errors.WithMessage(err, "fail to regiester new Peer")
	}

	if err := newPeer.Enroll(mspClient, false); err != nil {
		return &CaUser{}, err
	}
	if err := newPeer.Enroll(mspClient, true); err != nil {
		return &CaUser{}, err
	}

	// 5. create peer entity
	global.Logger.Info("peer starts creating")
	if err := kubernetes.NewPeer(newPeer.NetworkID, newPeer.OrganizationID, newPeer.ID).AwaitableCreate(); err != nil {
		return &CaUser{}, err
	}
	global.Logger.Info("peer has been created synchronously")

	return newPeer, err
}

func FindOrganizationByID(orgID int) (*Organization, error) {
	var orgs []Organization
	if err := global.DB.Where("id = ?", orgID).Find(&orgs).Error; err != nil {
		return &Organization{}, err
	}
	if len(orgs) < 0 {
		return &Organization{}, errors.New("no such org")
	}
	return &orgs[0], nil
}

func (org *Organization) UpdateStatus(status string) error {
	return global.DB.Model(&Organization{}).Where("id = ?", org.ID).Update("status", status).Error
}

// copy file
func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}

	defer destination.Close()
	return io.Copy(destination, source)
}