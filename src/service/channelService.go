package service

import (
	"encoding/base64"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"mictract/config"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"mictract/service/factory/sdk"
	"os"
	"path"
	"path/filepath"
	"text/template"
)

type ChannelService struct {
	ch *model.Channel
}

func NewChannelService(ch *model.Channel) *ChannelService {
	return &ChannelService{
		ch: ch,
	}
}

func (cSvc *ChannelService)GetAllAdminSigningIdentities() ([]msp.SigningIdentity, error) {
	global.Logger.Info("[[Get all admin signing identites in channel]]")
	orgs, err := dao.FindAllOrganizationsInChannel(cSvc.ch)
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

func (cSvc *ChannelService)CreateChannel(ordererURL string) error {
	global.Logger.Info("[channel is creating]")
	defer global.Logger.Info("[channel is creating] done!")

	channelConfigTxPath := filepath.Join(
		config.LOCAL_BASE_PATH,
		model.GetNetworkNameByID(cSvc.ch.NetworkID),
		fmt.Sprintf("%s.tx", cSvc.ch.GetName()))

	orgID := cSvc.ch.OrganizationIDs[0]
	org, err := dao.FindOrganizationByID(orgID)
	if err != nil {
		return err
	}

	// 1. get signs
	global.Logger.Info("1. Obtaining admin signature...")
	adminIdentity, err := NewOrganizationService(org).GetAdminSigningIdentity()
		//netSvc.GetAllAdminSigningIdentities()
	if err != nil {
		return errors.WithMessage(err, "fail to get all SigningIdentities")
	}

	// 2. generate req
	req := resmgmt.SaveChannelRequest{
		ChannelID: cSvc.ch.GetName(),
		ChannelConfigPath: channelConfigTxPath,
		SigningIdentities: []msp.SigningIdentity{adminIdentity},
	}

	// 3. get rc
	adminUser, err := dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		return err
	}

	rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		return errors.WithMessage(err, "fail to get rc")
	}

	// 4. submitting
	global.Logger.Info("2. Submitting to create channel transaction...")
	_, err = rc.SaveChannel(
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(ordererURL))

	return err
}


func (cSvc *ChannelService)GetSysChannelConfig() ([]byte, error) {
	global.Logger.Info("[[get system-channel config]]")

	orderers, err := dao.FindAllOrderersInNetwork(cSvc.ch.NetworkID)
	if err != nil {
		return []byte{}, err
	}
	ordOrg, err := dao.FindOrdererOrganizationInNetwork(cSvc.ch.NetworkID)
	if err != nil {
		return []byte{}, err
	}
	ordAdmin, err := dao.FindSystemUserInOrganization(ordOrg.ID)
	if err != nil {
		return []byte{}, err
	}

	rc, err := sdk.NewSDKClientFactory().
		NewResmgmtClient(ordAdmin)
	if err != nil {
		return []byte{}, errors.WithMessage(err, "fail to get rc")
	}

	cfg, err := rc.QueryConfigBlockFromOrderer(
		"system-channel",
		resmgmt.WithOrdererEndpoint(orderers[0].GetName()))
	if err != nil {
		return []byte{}, errors.WithMessage(err, "fail to query system-channel config")
	}

	return proto.Marshal(cfg)
}

func (cSvc *ChannelService) GetChannelConfig() ([]byte, error) {
	global.Logger.Info("[[get channel config]]")
	if cSvc.ch.ID == -1 {
		return nil, errors.New("please call GetSysChannelConfig")
	}

	orgID := cSvc.ch.OrganizationIDs[0]

	adminUser, err := dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		return []byte{}, err
	}

	peers, err := dao.FindAllPeersInOrganization(orgID)
	if err != nil {
		return []byte{}, err
	}

	sdkf := sdk.NewSDKClientFactory()
	ledgerClient, err := sdkf.NewLedgerClient(adminUser, cSvc.ch)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get ledgerClient")
	}

	cfg, err := ledgerClient.QueryConfigBlock(ledger.WithTargetEndpoints(peers[0].GetName()))
	if err != nil {
		return nil, errors.WithMessage(err, "fail to query config")
	}

	return proto.Marshal(cfg)
}

// Don't use for system-channel
func (cSvc *ChannelService) GetChannelInfo() (*fab.BlockchainInfoResponse, error) {
	global.Logger.Info("[[get channel info]]")
	var err error
	if len(cSvc.ch.OrganizationIDs) < 1 {
		return &fab.BlockchainInfoResponse{}, errors.New("No organization in the channel")
	}

	for _, orgID := range cSvc.ch.OrganizationIDs {
		adminUser, err := dao.FindSystemUserInOrganization(orgID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}
		lc, err := sdk.NewSDKClientFactory().NewLedgerClient(adminUser, cSvc.ch)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}

		peers, err := dao.FindAllPeersInOrganization(orgID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}

		for _, peer := range peers {
			var ret *fab.BlockchainInfoResponse
			ret, err = lc.QueryInfo(ledger.WithTargetEndpoints(peer.GetName()))
			if err != nil {
				global.Logger.Error("", zap.Error(err))
			}
			// !!!出口
			return ret, nil
		}
	}
	return &fab.BlockchainInfoResponse{}, err
}

// Don't use for system-channel
func (cSvc *ChannelService) GetBlock(blockID uint64) (*common.Block, error) {
	global.Logger.Info("[[get block]]")
	orgID := cSvc.ch.OrganizationIDs[0]
	adminUser, err := dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
		return &common.Block{}, err
	}
	peers, err := dao.FindAllPeersInOrganization(orgID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
		return &common.Block{}, err
	}

	lc, err := sdk.NewSDKClientFactory().NewLedgerClient(adminUser, cSvc.ch)
	if err != nil {
		return &common.Block{}, err
	}
	return lc.QueryBlock(blockID, ledger.WithTargetEndpoints(peers[0].GetName()))
}

func (cSvc *ChannelService)GetAndStoreConfig() error {
	global.Logger.Info("[[get and store config]]")

	// 1. get
	bt := []byte{}
	var err error
	if cSvc.ch.ID != -1 {
		bt, err = cSvc.GetChannelConfig()
		if err != nil {
			return err
		}
	} else {
		bt, err = cSvc.GetSysChannelConfig()
		if err != nil {
			return err
		}
	}

	// 2. store
	f, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "config_block.pb"))
	if err != nil {
		return err
	}
	_, err = f.Write(bt)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func (cSvc *ChannelService)updateConfig(signs []msp.SigningIdentity) error {
	global.Logger.Info("[[update config]]")
	envelopeFile, err := os.Open(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()

	orgID := cSvc.ch.OrganizationIDs[0]

	adminUser, err := dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		return err
	}

	orderers, err := dao.FindAllOrderersInNetwork(cSvc.ch.NetworkID)
	if err != nil {
		return err
	}

	req := resmgmt.SaveChannelRequest{
		ChannelID:         cSvc.ch.GetName(),
		ChannelConfig:     envelopeFile,
		SigningIdentities: signs,
	}
	resmgmtClient, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		return err
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

// AddOrg uses the existing organization's certificate to update the configuration of the channel
func (cSvc *ChannelService) AddOrg(orgID int) error {
	global.Logger.Info(fmt.Sprintf("[Add org%d to channel%d]", orgID, cSvc.ch.ID))
	defer global.Logger.Info(fmt.Sprintf("[Add org%d to channel%d] done!", orgID, cSvc.ch.ID))

	org, err := dao.FindOrganizationByID(orgID)
	if err != nil {
		return err
	}

	// 1. Obtaining channel config
	global.Logger.Info("1. Obtaining channel config")
	if err := cSvc.GetAndStoreConfig(); err != nil {
		return err
	}

	// 2. generate configtx.yaml
	global.Logger.Info("2. generate configtx.yaml")
	configtxFile, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "configtx.yaml"))
	if err != nil {
		return errors.WithMessage(err, "fail to open configtx.yaml")
	}

	_, err = configtxFile.WriteString(org.GetConfigtxFile())
	if err != nil {
		return errors.WithMessage(err, "fail to write configtx.yaml")
	}

	// 3. call addorg.sh to generate org_update_in_envelope.pb
	global.Logger.Info("3. call addorg.sh to generate org_update_in_envelope.pb")
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrg",
		cSvc.ch.GetName(),
		org.GetMSPID(),)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// 4. sign for org_update_in_envelope.pb
	global.Logger.Info("4. sign for org_update_in_envelope.pb")
	signs, err := cSvc.GetAllAdminSigningIdentities()
		//NewNetworkService(net).GetAllAdminSigningIdentities()
	if err != nil {
		return err
	}

	// 5. update org_update_in_envelope.pb
	global.Logger.Info("5. Update channel config...")
	if err := cSvc.updateConfig(signs); err != nil {
		return err
	}

	return dao.UpdateOrgIDs(cSvc.ch.ID, orgID)
}

func (cSvc *ChannelService)UpdateAnchors(orgID int) error {
	global.Logger.Info("[[update anchor]]")
	// 不要让用户自定义了，让所有peer都成为锚节点
	if orgID <= 0 {
		return errors.New(fmt.Sprintf("org ID is incorrect. ID: %d", orgID))
	}

	org, err := dao.FindOrganizationByID(orgID)
	if err != nil {
		return err
	}
	net, err := dao.FindNetworkByID(cSvc.ch.NetworkID)
	if err != nil {
		return err
	}

	// generate config_block.pb
	if err := cSvc.GetAndStoreConfig(); err != nil {
		return err
	}

	// generate anchors.json
	peers, err := dao.FindAllPeersInOrganization(orgID)
	if err != nil {
		return err
	}

	st := `{"mod_policy":"Admins","value":{"anchor_peers":[`
	for _, peer := range peers {
		st += `{"host":"` + peer.GetURL() + `","port":7051},`
	}
	//st += `{"host":"` + "lilingj.github.io" + `","port":7051},`
	// jq这个坑货，多一个逗号就解析不出来
	st = st[:(len(st) - 1)]
	st += `]},"version":"0"}`
	f, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "anchors.json"))
	if err != nil {
		return err
	}
	if _, err = f.WriteString(st); err != nil {
		return err
	}
	f.Close()

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	//cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "updateAnchors", c.Name, org.MSPID)
	//output, err := cmd.CombinedOutput()
	//global.Logger.Info(string(output))
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"updateAnchors",
		cSvc.ch.GetName(),
		org.GetMSPID())
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs, err := NewNetworkService(net).GetAllAdminSigningIdentities()
	if err != nil {
		return err
	}

	// update org_update_in_envelope.pb
	return cSvc.updateConfig(signs)
}

// AddOrderers
// eg: GetSystemChannel(n.ID).AddOrderers(Order{fmt.Sprintf("orderer%d.net%d.com", len(n.Orderers) + 1, n.ID)})
func (cSvc *ChannelService)AddOrderers(orderer model.CaUser) error {
	global.Logger.Info("[[Add Orderer to channel]]")
	if cSvc.ch.ID != -1 {
		return errors.New("only for system-channel")
	}

	// generate config_block.pb
	if err := cSvc.GetAndStoreConfig(); err != nil {
		return err
	}

	// generate ord1.json
	orderers, err := dao.FindAllOrderersInNetwork(cSvc.ch.NetworkID)
	if err != nil {
		return err
	}
	net, err := dao.FindNetworkByID(cSvc.ch.NetworkID)
	if err != nil {
		return err
	}


	st := `["`
	for _, orderer := range orderers {
		tlscert, err := dao.FindCertByUserID(orderer.ID, true)
		if err != nil {
			return err
		}
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert.Certification)) +
			`","host":"` + orderer.GetURL() +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert.Certification)) + `"},`
	}
	st += "]"
	f1, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "ord1.json"))
	if err != nil {
		return err
	}
	if _, err = f1.WriteString(st); err != nil {
		return err
	}
	f1.Close()

	// generate ord2.json
	st = `[`
	for _, orderer := range orderers {
		st += `"` + orderer.GetURL() + `",`
	}
	st += "]"
	f2, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "ord2.json"))
	if err != nil {
		return err
	}
	if _, err = f2.WriteString(st); err != nil {
		return err
	}
	f2.Close()

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	//cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "addOrderers", c.Name)
	//output, err := cmd.CombinedOutput()
	//global.Logger.Info(string(output))
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrderers",
		cSvc.ch.GetName(),
	)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs, err := NewNetworkService(net).GetAllAdminSigningIdentities()
	if err != nil {
		return err
	}

	// update org_update_in_envelope.pb
	return cSvc.updateConfig(signs)
}

// 渲染一个通道，只包含通道中第一个org
func (cSvc *ChannelService) RenderConfigtx() error {
	global.Logger.Info("[[render config tx]]")
	templ := template.Must(template.ParseFiles(path.Join(config.LOCAL_MOUNT_PATH, "channel.yaml.tpl")))

	filename := fmt.Sprintf("/mictract/networks/net%d/configtx.yaml", cSvc.ch.NetworkID)
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	orgs, err := dao.FindAllOrganizationsInChannel(cSvc.ch)
	if err != nil {
		return err
	}

	return templ.Execute(writer, orgs[0])
}
