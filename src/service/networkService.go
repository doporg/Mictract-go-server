package service

import (
	"encoding/base64"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	mConfig "mictract/config"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
	"os"
	"path/filepath"
	"time"
)

type NetworkService struct {
	net *model.Network
}

func NewNetworkService(net *model.Network) *NetworkService {
	return &NetworkService{
		net: net,
	}
}

// Deploy method is just creating a basic network containing only ordererorg(including 1 orderer)
// The basic network is built to make `configtx.yaml` file simple enough to create the genesis block.
//
// net := factory.NewNetworkFactory().NewNetwork(nickname, consensus)
// netSvc := NewNetworkService(net)
// netSvc.Deploy()
func (ns *NetworkService)Deploy() error {
	global.Logger.Info(fmt.Sprintf("[Deploy %s]", ns.net.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Deploy %s] done!", ns.net.GetName()))
	global.Logger.Info("Deploy method is just creating a basic network containing only ordererorg(including 1 orderer)")

	tools := kubernetes.Tools{}

	// 1. Start orderer ca and register system users and orderer nodes
	global.Logger.Info("1. Start orderer ca and register system users and orderer nodes")
	ordererOrg, err := factory.NewOrganizationFactory().NewOrdererOrganization(ns.net.ID, "ordererorg")
	ordererOrgSvc := NewOrganizationService(ordererOrg)
	if err != nil {
		return err
	}
	if err := ordererOrgSvc.CreateBasicOrganizationEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg", zap.Error(err))
	}

	// 2. render configtx.yaml
	global.Logger.Info("2. Render configtx.yaml")
	orderers, err := dao.FindAllOrderersInNetwork(ns.net.ID)
	if err != nil {
		return err
	}
	if err = orderers[0].RenderConfigtx(); err != nil {
		return err
	}

	// 3. generate the genesis block
	global.Logger.Info("3. Generate the genesis block")
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", ns.net.ID),
		"-profile", "Genesis",
		"-channelID", "system-channel",
		"-outputBlock", fmt.Sprintf("/mictract/networks/net%d/genesis.block", ns.net.ID),
	)
	if err != nil {
		return err
	}

	// 4. start one orderer
	global.Logger.Info("4. start one orderer")
	if err := ordererOrgSvc.CreateNodeEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg's node", zap.Error(err))
	}

	return nil
}

// If the memory overflows, the problem may lie here
func (ns *NetworkService)deleteGlobalSvc() {
	// delete sdk and AdminSigns
	orgs, _ := dao.FindAllOrganizationsInNetwork(ns.net.ID)
	for _, org := range orgs {
		orgSDK, ok := global.SDKs[org.GetName()]
		if ok {
			orgSDK.Close()
			delete(global.SDKs, org.GetName())
		}

		adminUser, _ := dao.FindSystemUserInOrganization(org.ID)
		delete(global.AdminSigns, adminUser.GetName())
	}
	netSDK, ok := global.SDKs[ns.net.GetName()]
	if ok {
		netSDK.Close()
		delete(global.SDKs, ns.net.GetName())
	}

	// delete adminSign

}

func (ns *NetworkService)Delete() error {
	global.Logger.Info(fmt.Sprintf("[Delete %s]", ns.net.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Delete %s] done!", ns.net.GetName()))

	var orgs []model.Organization
	var ccs  []model.Chaincode
	var err  error

	// 0.
	ns.deleteGlobalSvc()

	// 1. remove organizations entity and close sdk
	global.Logger.Info("1. remove organizations entity")
	orgs, err = dao.FindAllOrganizationsInNetwork(ns.net.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, org := range orgs {
		NewOrganizationService(&org).RemoveAllEntity()
	}

	// 2. remove chaincode entity
	global.Logger.Info("2. remove chaincode entity")
	if ccs, err = dao.FindAllChaincodes(); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, cc := range ccs {
		NewChaincodeService(&cc).RemoveEntity()
	}

	// 3.  Delete all database records related to the network
	// TODO: Cascade delete
	global.Logger.Info("3. Delete all database records related to the network")
	// 3.1 delete from organizations where network_id = id
	global.Logger.Info("3.1 delete from organizations where network_id = id")
	if err := dao.DeleteAllOrganizationsInNetwork(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	// 3.2 delete from ca_users where network_id = id
	global.Logger.Info("3.2 delete from ca_users where network_id = id")
	if err := dao.DeleteAllCaUserInNetwork(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	// 3.3 delete from chaincode where network_id = id
	global.Logger.Info("3.3 delete from chaincode where network_id = id")
	if err:= dao.DeleteAllChaincodesInNetwork(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	// 3.4 delete from channels where network_id = id
	global.Logger.Info("3.4 delete from channels where network_id = id")
	if err := dao.DeleteAllChannelsInNetwork(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	// 3.5 delete from networks where id = id
	global.Logger.Info("3.5 delete from networks where id = id")
	if err := dao.DeleteNetworkByID(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	// 3.6 delete from network where network_id = id
	global.Logger.Info("3.6 delete from network where network_id = id")
	if err := dao.DeleteAllCertificationsInNetwork(ns.net.ID); err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	// 4. TODO: RemoveAllFile
	// os.RemoveAll("/mictract/network/net%d")

	return nil
}

func (ns *NetworkService)GetAllAdminSigningIdentities() ([]msp.SigningIdentity, error) {
	global.Logger.Info("[[Get all admin signing identites]]")
	orgs, err := dao.FindAllOrganizationsInNetwork(ns.net.ID)
	if err != nil {
		return []msp.SigningIdentity{}, err
	}

	signs := []msp.SigningIdentity{}
	for _, org := range orgs {
		sign, err := NewOrganizationService(&org).GetAdminSigningIdentity()
		if err != nil {
			return signs, err
		}
		signs = append(signs, sign)
	}

	return signs, nil
}

func (ns *NetworkService)AddOrgToConsortium(orgID int) error {
	global.Logger.Info(fmt.Sprintf("[Add org%d to Consortium(Write to system-channel)]", orgID))
	defer global.Logger.Info(fmt.Sprintf("[Add org%d to Consortium(Write to system-channel)] done!", orgID))

	orderers, err := dao.FindAllOrderersInNetwork(ns.net.ID)
	if err != nil {
		return err
	}
	org, err := dao.FindOrganizationByID(orgID)
	if err != nil {
		return err
	}
	ordOrg, err := dao.FindOrdererOrganizationInNetwork(ns.net.ID)
	if err != nil {
		return err
	}
	ordAdminUser, err := dao.FindSystemUserInOrganization(ordOrg.ID)
	if err!= nil {
		return err
	}

	// 1. Obtaining channel config
	global.Logger.Info("1. Obtaining channel config")
	sysch := factory.NewChannelFactory().NewSystemChannel(ns.net.ID)
	if err :=NewChannelService(sysch).GetAndStoreConfig(); err != nil {
		return err
	}

	// 2. generate configtx.yaml
	global.Logger.Info("2. Generate configtx.yaml")
	configtxFile, err := os.Create(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "configtx.yaml"))
	if err != nil {
		return errors.WithMessage(err, "fail to open configtx.yaml")
	}
	_, err = configtxFile.WriteString(org.GetConfigtxFile())
	if err != nil {
		return errors.WithMessage(err, "fail to write configtx.yaml")
	}

	// 3. generate org_update_in_envelope.pb
	global.Logger.Info("3. Generate org_update_in_envelope.pb")
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrgToConsortium",
		"system-channel",
		org.GetMSPID())
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	sign, err := NewOrganizationService(ordOrg).GetAdminSigningIdentity()
	if err != nil {
		return errors.WithMessage(err, "ordererAdmin fail to sign")
	}

	// 4. update org_update_in_envelope.pb
	global.Logger.Info("4. update org_update_in_envelope.pb")
	envelopeFile, err := os.Open(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()

	rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(ordAdminUser)
	if err != nil {
		return err
	}

	req := resmgmt.SaveChannelRequest{
		ChannelID:         "system-channel",
		ChannelConfig:     envelopeFile,
		SigningIdentities: []msp.SigningIdentity{sign},
	}
	_, err = rc.SaveChannel(
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(orderers[0].GetName()))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}

	return nil
}

// AddOrg creates an organizational entity
func (ns *NetworkService) AddOrg(nickname string) (*model.Organization, error) {
	global.Logger.Info(fmt.Sprintf("[Add new org to %s]", ns.net.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Add new org to %s] done!", ns.net.GetName()))

	// 1. new model.organization
	global.Logger.Info("1. new model.organization")
	org, err := factory.NewOrganizationFactory().NewOrganization(ns.net.ID, nickname)
	if err != nil {
		return &model.Organization{}, err
	}
	orgSvc := NewOrganizationService(org)

	// 2. create org entity(ca, peer) and system user
	global.Logger.Info(fmt.Sprintf("2. create %s entity(ca, peer) and system user", org.GetName()))
	if err := orgSvc.CreateBasicOrganizationEntity(); err != nil {
		dao.UpdateOrganizationStatusByID(org.ID, enum.StatusError)
		return &model.Organization{}, err
	}
	if err := orgSvc.CreateNodeEntity(); err != nil {
		dao.UpdateOrganizationStatusByID(org.ID, enum.StatusError)
		return &model.Organization{}, err
	}

	// 3. Update organization to Consortium
	global.Logger.Info("3. Update organization to Consortium")
	if err := ns.AddOrgToConsortium(org.ID); err != nil {
		dao.UpdateOrganizationStatusByID(org.ID, enum.StatusError)
		return &model.Organization{}, err
	}

	return org, nil
}

func (ns *NetworkService) AddChannel(orgIDs []int, nickname string) (*model.Channel, error) {
	global.Logger.Info(fmt.Sprintf("[Add new channel to %s]", ns.net.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Add new channel to %s] done!", ns.net.GetName()))

	// 1. new model.Channel
	global.Logger.Info("1. new model.Channel")
	ch, err := factory.NewChannelFactory().NewChannel(ns.net.ID, nickname, orgIDs[0:1])
	if err != nil {
		return ch, err
	}
	orderers, err := dao.FindAllOrderersInNetwork(ns.net.ID)
	if err != nil {
		return ch, err
	}

	chSvc := NewChannelService(ch)

	// 2. render configtx.yaml
	global.Logger.Info("2. render configtx.yaml")
	if err := chSvc.RenderConfigtx(); err != nil {
		return ch, errors.WithMessage(err, "fail to render configtx.yaml")
	}

	// 3. generate channel.tx
	global.Logger.Info(fmt.Sprintf("3. generate %s.tx", ch.GetName()))
	tools := kubernetes.Tools{}
	global.Logger.Info("generate a default channel")
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", ns.net.ID),
		"-profile", "NewChannel",
		"-channelID", ch.GetName(),
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel%d.tx", ns.net.ID, ch.ID),
	)
	if err != nil {
		return ch, err
	}

	// 4. subbmit create channel tx
	global.Logger.Info("4. subbmit create channel tx")
	if err := chSvc.CreateChannel(orderers[0].GetName()); err != nil {
		return ch, err
	}

	// 5. all peer in first org join channel
	global.Logger.Info("5. All peer in first org join channel")
	peers, err := dao.FindAllPeersInOrganization(orgIDs[0])
	for _, peer := range peers {
		global.Logger.Info(fmt.Sprintf("%s join channel", peer.GetName()))
		if err := NewCaUserService(&peer).JoinChannel(ch.ID, orderers[0].GetName()); err != nil {
			global.Logger.Error("", zap.Error(err))
		}
	}

	// 5.1 updata anchors (the first org in channel)
	global.Logger.Info("5.1 updata anchors (the first org in channel)")
	if err := chSvc.UpdateAnchors(ch.OrganizationIDs[0]); err != nil {
		return ch, err
	}

	// WTF? If you don’t wait, you will get an outdated configuration and return an error
	time.Sleep(5 * time.Second)

	// 6. Dynamically add remaining organizations to the channel
	global.Logger.Info("6. Dynamically add remaining organizations to the channel")
	for i := 1; i < len(orgIDs); i++ {
		global.Logger.Info(fmt.Sprintf(" ┗ add org%d to channel%d", orgIDs[i], ch.ID))
		// WTF? If you don’t wait, you will get an outdated configuration and return an error
		time.Sleep(5 * time.Second)
		if err := chSvc.AddOrg(orgIDs[i]); err != nil {
			return ch, err
		}
	}
	// bug: orderer处理更新配置文件需要时间, 加入通道操作在orderer还每处理完通道配置更新交易就提交上去，
	//      会导致身份识别错误。注意这个玄学问题
	time.Sleep(5 * time.Second)

	// 7. Dynamically join remaining peer to the channel
	/*
		"Tried joining channel channel3 but our org( org2MSP ), isn't among the orgs of the channel: [org3MSP] , aborting."
		If the above error occurs in the peer container
		just fuck blockchain
	*/
	global.Logger.Info("7. Dynamically join remaining peer to the channel")
	for i := 1; i < len(orgIDs); i++ {
		global.Logger.Info(fmt.Sprintf(" update anchors(org%d)", orgIDs[i]))
		if err := chSvc.UpdateAnchors(orgIDs[i]); err != nil {
			return ch, err
		}

		peers, err := dao.FindAllPeersInOrganization(orgIDs[i])
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}
		for _, peer := range peers {
			go func(peer *model.CaUser){
				global.Logger.Info(fmt.Sprintf("%s join channel", peer.GetName()))
				time.Sleep(5 * time.Second)
				if err := NewCaUserService(peer).JoinChannel(
					ch.ID,
					orderers[0].GetName()); err != nil {
					global.Logger.Error(fmt.Sprintf("%s fail to join channel%d", peer.GetName(), ch.ID), zap.Error(err))
				}
			}(&peer)
		}
	}

	return ch, nil
}

// AddOrderers
// consensus must be "etcdraft"
func (ns *NetworkService)AddOrderersToSystemChannel() error {
	global.Logger.Info(fmt.Sprintf("[Add Orderer to %s(system-channel)]", ns.net.GetName()))
	defer global.Logger.Info(fmt.Sprintf("[Add Orderer to %s(system-channel)] done!", ns.net.GetName()))

	ch := factory.NewChannelFactory().NewSystemChannel(ns.net.ID)
	chSvc := NewChannelService(ch)
	orderers, err := dao.FindAllOrderersInNetwork(ns.net.ID)
	if err != nil {
		return err
	}
	ordOrg, err := dao.FindOrdererOrganizationInNetwork(ns.net.ID)
	if err != nil {
		return err
	}
	adminUser, err := dao.FindSystemUserInOrganization(ordOrg.ID)
	if err != nil {
		return err
	}

	// 1. new model.CaUser (orderer)
	global.Logger.Info("1. new model.CaUser (orderer)")
	user, err := factory.NewCaUserFactory().NewOrdererCaUser(ordOrg.ID, ns.net.ID, "orderer1")
	if err != nil {
		return err
	}
	userSvc := NewCaUserService(user)

	// 2. generate config_block.pb
	global.Logger.Info("2. Get and Store system-channel config")
	if err := chSvc.GetAndStoreConfig(); err != nil {
		return err
	}

	// 3. regiester new orderer
	global.Logger.Info("3. regiester new orderer")
	mspClient, err := sdk.NewSDKClientFactory().NewMSPClient(ordOrg)
	if err != nil {
		return err
	}
	if err := userSvc.Register(mspClient); err != nil {
		return err
	}

	// 4. enroll
	global.Logger.Info("4. Enroll new orderer")
	if err := userSvc.Enroll(mspClient, true); err != nil {
		return err
	}
	if err := userSvc.Enroll(mspClient, false); err != nil {
		return err
	}

	// 5. create orderer entity
	global.Logger.Info("5. create orderer entity")
	if err := kubernetes.NewOrderer(ns.net.ID, user.ID).AwaitableCreate(); err != nil {
		return err
	}

	// 6. generate ord1.json
	global.Logger.Info("6. generate ord1.json")
	st := `[`
	for _, orderer := range orderers {
		tlscert, _ := dao.FindCertByUserID(orderer.ID, true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert.Certification)) +
			`","host":"` + orderer.GetURL() +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert.Certification)) + `"},`
	}
	st = st[:len(st) - 1]
	st += "]"
	f1, err := os.Create(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "ord1.json"))
	if err != nil {
		return err
	}
	if _, err = f1.WriteString(st); err != nil {
		return err
	}
	f1.Close()

	// 7. generate ord2.json
	global.Logger.Info("7. generate ord2.json")
	st = `[`
	for _, orderer := range orderers {
		st += `"` + orderer.GetURL() + `",`
	}
	st = st[:len(st) - 1]
	st += "]"
	f2, err := os.Create(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "ord2.json"))
	if err != nil {
		return err
	}
	if _, err = f2.WriteString(st); err != nil {
		return err
	}
	f2.Close()

	// 8. call addorg.sh to generate org_update_in_envelope.pb
	global.Logger.Info("8. call addorg.sh to generate org_update_in_envelope.pb")
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrderers",
		"system-channel",
	)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// 9. sign for org_update_in_envelope.pb and update it
	global.Logger.Info("9. sign for org_update_in_envelope.pb and update it")
	signs := []msp.SigningIdentity{}
	adminIdentity, err := mspClient.GetSigningIdentity(adminUser.GetName())
	if err != nil {
		return errors.WithMessage(err, "ordererAdmin fail to sign")
	}
	signs = append(signs, adminIdentity)

	// 10. update org_update_in_envelope.pb
	global.Logger.Info("10. update org_update_in_envelope.pb...")
	envelopeFile, err := os.Open(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()

	resmgmtClient, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		return err
	}

	req := resmgmt.SaveChannelRequest{
		ChannelID:         "system-channel",
		ChannelConfig:     envelopeFile,
		SigningIdentities: signs,
	}
	_, err = resmgmtClient.SaveChannel(
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(orderers[0].GetName()))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}
	return nil
}