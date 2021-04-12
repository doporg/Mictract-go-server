package service

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
	"os"
	"path/filepath"
	"time"
)

type OrganizationService struct {
	org *model.Organization
}

func NewOrganizationService(org *model.Organization) *OrganizationService {
	return &OrganizationService{
		org: org,
	}
}

func (orgSvc *OrganizationService) GetAdminSigningIdentity() (msp.SigningIdentity, error){
	global.Logger.Info(fmt.Sprintf("[Get %s admin signing identity]", orgSvc.org.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Get %s admin signing identity] done!", orgSvc.org.GetName()))

	// 1. get admin user
	adminUsername, err := dao.FindSystemUserInOrganization(orgSvc.org.ID)
	if err != nil {
		return nil, err
	}

	// 2. get msp client
	mspClient, err := sdk.NewSDKClientFactory().NewMSPClient(orgSvc.org)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get mspClient "+orgSvc.org.GetName())
	}

	// 3. get admin signing identity
	adminIdentity, err := mspClient.GetSigningIdentity(adminUsername.GetName())
	if err != nil {
		return nil, errors.WithMessage(err, orgSvc.org.GetName()+"fail to sign")
	}
	return adminIdentity, nil
}

func (orgSvc *OrganizationService) GenerateOrgMSP() error {
	global.Logger.Info(fmt.Sprintf("[[generate %s msp]]", orgSvc.org.GetName()))

	basePath := orgSvc.org.GetMSPDir()

	// cacerts/ca-cert.pem
	if err := os.MkdirAll(filepath.Join(basePath, "cacerts"), os.ModePerm); err != nil {
		return err
	}
	if _, err := copy(filepath.Join(basePath, "..", "ca", "ca-cert.pem"), filepath.Join(basePath, "cacerts", "ca-cert.pem")); err != nil {
		return err
	}

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


	// 2. insert into db
	cacert, err := ioutil.ReadFile(filepath.Join(orgSvc.org.GetMSPDir(), "cacerts", "ca-cert.pem"))
	if err != nil {
		return errors.WithMessage(err, "fail to read ca-cert.pem")
	}
	if _, err := factory.NewCertificationFactory().NewCACertification(orgSvc.org, string(cacert), ""); err != nil {
		return errors.WithMessage(err, "fail to insert cert into db")
	}

	return nil
}

// CreateBasicOrganizationEntity starts a CA node,
// and registers the node certificate and admin certificates
func (orgSvc *OrganizationService)CreateBasicOrganizationEntity() error {
	global.Logger.Info(fmt.Sprintf("[create basic %s entity]", orgSvc.org.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[create basic %s entity] done!", orgSvc.org.GetName()))

	// 1. create ca pod
	global.Logger.Info("1. create ca pod synchronously")
	ca := kubernetes.NewOrdererCA(orgSvc.org.NetworkID)
	if !orgSvc.org.IsOrdererOrganization() {
		ca = kubernetes.NewPeerCA(orgSvc.org.NetworkID, orgSvc.org.ID)
	}
	if err := ca.AwaitableCreate(); err != nil {
		return err
	}

	// 2. Wait for the ca program to start
	global.Logger.Info("2. Wait for the ca program to start")
	time.Sleep(15 * time.Second)

	// 3. 从ca的挂载目录里取出ca证书，构建组织msp, insert cert into db
	global.Logger.Info("3. The msp of the organization is being built")
	if err := orgSvc.GenerateOrgMSP(); err != nil {
		return err
	}

	// 4. register Admin1@org%d.net%d.com or Admin1@net%d.com
	global.Logger.Info("4. Registering users(1 system-user 1 peer or orderer)...")
	mspClient, err := sdk.NewSDKClientFactory().NewMSPClient(orgSvc.org)
	if err != nil {
		return err
	}
	newUser, err := factory.NewCaUserFactory().NewAdminCaUser(
		orgSvc.org.ID,
		orgSvc.org.NetworkID,
		"system-user",
		"admin1pw",
		orgSvc.org.IsOrdererOrg)
	if err != nil {
		return err
	}
	users := []*model.CaUser{newUser}
	if orgSvc.org.IsOrdererOrganization() {
		newUser, err := factory.NewCaUserFactory().NewOrdererCaUser(
			orgSvc.org.ID,
			orgSvc.org.NetworkID,
			"orderer1pw")
		if err != nil {
			return err
		}
		users = append(users, newUser)
	} else {
		newUser, err := factory.NewCaUserFactory().NewPeerCaUser(
			orgSvc.org.ID,
			orgSvc.org.NetworkID,
			"peer1pw")
		if err != nil {
			return err
		}
		users = append(users, newUser)
	}

	for _, u := range users {
		if err := NewCaUserService(u).Register(mspClient); err != nil {
			return err
		}
	}

	// 5. enroll to build msp and tls directories
	global.Logger.Info("5. User certificate is being stored...")
	for _, u := range users {
		// msp
		if err := NewCaUserService(u).Enroll(mspClient, false); err != nil {
			return err
		}
		// tls
		if err := NewCaUserService(u).Enroll(mspClient, true); err != nil {
			return err
		}
	}

	return nil
}

// CreateNodeEntity creates a node entity and starts the peer or orderer node
// in the organization through the existing certificate
func (orgSvc *OrganizationService) CreateNodeEntity() error {
	global.Logger.Info(fmt.Sprintf("[create %s node entity]", orgSvc.org.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[create %d node entity] done!", orgSvc.org.GetName()))

	if orgSvc.org.IsOrdererOrganization() {
		global.Logger.Info("orderer starts creating")
		// 此时应该只有一个
		orderers, err := dao.FindAllOrderersInNetwork(orgSvc.org.NetworkID)
		if err != nil {
			return err
		}
		if err := kubernetes.NewOrderer(orgSvc.org.NetworkID, orderers[0].ID).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("orderer has been created synchronously")
	} else {
		global.Logger.Info("peer starts creating")
		// 此时应该只有一个
		peers, err := dao.FindAllPeersInOrganization(orgSvc.org.ID)
		if err != nil {
			return err
		}
		if err := kubernetes.NewPeer(orgSvc.org.NetworkID, orgSvc.org.ID, peers[0].ID).AwaitableCreate(); err != nil {
			return err
		}
		global.Logger.Info("peer has been created synchronously")
	}

	return nil
}

func (orgSvc *OrganizationService) RemoveAllEntity() {
	global.Logger.Info(fmt.Sprintf("[remove %s entity]", orgSvc.org.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[remove %s entity] done!", orgSvc.org.GetName()))

	if !orgSvc.org.IsOrdererOrganization() {
		global.Logger.Info("Remove PeerCA...")
		// 1. get users
		users, err := dao.FindAllPeersInOrganization(orgSvc.org.ID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
		}

		// 2. remove ca entity
		kubernetes.NewPeerCA(orgSvc.org.NetworkID, orgSvc.org.ID).Delete()

		// 3. remove peer entity
		for _, peer := range users {
			global.Logger.Info("Remove " + peer.GetName() + "...")
			kubernetes.NewPeer(peer.NetworkID, peer.OrganizationID, peer.ID).Delete()
		}
	} else {
		global.Logger.Info("Remove OrdererCA...")
		// 1. get users
		users, err := dao.FindAllOrderersInNetwork(orgSvc.org.NetworkID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			return
		}

		// 2. remove ca entity
		kubernetes.NewOrdererCA(orgSvc.org.NetworkID).Delete()

		// 3. remove orderer entity
		for _, orderer := range users {
			global.Logger.Info("Remove " + orderer.GetName() + "...")
			kubernetes.NewOrderer(orderer.NetworkID, orderer.ID).Delete()
		}
	}
}

func (orgSvc *OrganizationService)AddPeer() (*model.CaUser, error) {
	global.Logger.Info(fmt.Sprintf("[add new peer to %s]", orgSvc.org.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[add new peer to %s] done!", orgSvc.org.GetName()))
	// 1. check
	if orgSvc.org.IsOrdererOrganization() {
		return &model.CaUser{}, errors.New("Just for peer, not orderer")
	}

	// 3. get mspclient
	mspClient, err := sdk.NewSDKClientFactory().NewMSPClient(orgSvc.org)
	if err != nil {
		return &model.CaUser{}, err
	}

	// 4. register and enroll
	newPeer, err := factory.NewCaUserFactory().NewPeerCaUser(
		orgSvc.org.ID,
		orgSvc.org.NetworkID,
		"peer1pw")
	if err != nil {
		return &model.CaUser{}, err
	}
	if err := NewCaUserService(newPeer).Register(mspClient); err != nil {
		return &model.CaUser{}, errors.WithMessage(err, "fail to regiester new Peer")
	}

	if err := NewCaUserService(newPeer).Enroll(mspClient, false); err != nil {
		return &model.CaUser{}, err
	}
	if err := NewCaUserService(newPeer).Enroll(mspClient, true); err != nil {
		return &model.CaUser{}, err
	}

	// 5. create peer entity
	global.Logger.Info("peer starts creating")
	if err := kubernetes.NewPeer(newPeer.NetworkID, newPeer.OrganizationID, newPeer.ID).AwaitableCreate(); err != nil {
		return &model.CaUser{}, err
	}
	global.Logger.Info("peer has been created synchronously")

	return newPeer, err
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