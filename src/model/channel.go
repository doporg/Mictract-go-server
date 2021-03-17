package model

import (
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"go.uber.org/zap"
	"mictract/model/kubernetes"
	"path"
	"text/template"

	"mictract/config"
	"mictract/global"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
)

type Channel struct {
	ID            int 	`json:"id"`
	Name          string        `json:"name"`
	NetworkID   int        `json:"networkID"`
	Organizations Organizations `json:"organizations"`
	Orderers      Orders        `json:"orderers"`
}

type Channels []Channel

// 自定义数据字段所需实现的两个接口
func (channels *Channels) Scan(value interface{}) error {
	return scan(&channels, value)
}

func (channels Channels) Value() (driver.Value, error) {
	return value(channels)
}

func (channel *Channel) Scan(value interface{}) error {
	return scan(&channel, value)
}

func (channel Channel) Value() (driver.Value, error) {
	return value(channel)
}

// 防止传进来的channel对象不完整，比如更新组织对象时没有无法更新到channel对象，
// 所以从Nets中重新构造一个channel，保证信息最新
func GetChannelFromNets(channelID int, netID int) (*Channel, error) {
	net, err := GetNetworkfromNets(netID)
	if err != nil {
		return nil, err
	}

	if channelID == -1 {
		return nil, errors.New("Does not support system-channel, please call GetSystemChannel")
	}
	if len(net.Channels) < channelID || channelID < -1 || channelID == 0{
		return nil, errors.New("The channel does not exist in the network")
	}


	ret := net.Channels[channelID - 1]
	for i, org := range net.Channels[channelID - 1].Organizations {
		ret.Organizations[i] = net.Organizations[org.ID]
	}
	return &ret, nil
}

func (c *Channel) NewLedgerClient(username, orgname string) (*ledger.Client, error) {
	//sdk, ok := global.SDKs[c.NetworkName]
	if err := UpdateSDK(c.NetworkID); err != nil {
		return nil, err
	}
	sdk, err := GetSDKByNetWorkID(c.NetworkID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk ")
	}
	ledgerClient, err := ledger.New(sdk.ChannelContext(fmt.Sprintf("channel%d", c.ID), fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return nil, err
	}
	return ledgerClient, nil
}

func (c *Channel) NewResmgmtClient(username, orgname string) (*resmgmt.Client, error) {
	if err := UpdateSDK(c.NetworkID); err != nil {
		return nil, err
	}
	sdk, err := GetSDKByNetWorkID(c.NetworkID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk ")
	}
	resmgmtClient, err := resmgmt.New(sdk.Context(fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return nil, err
	}
	return resmgmtClient, nil
}

func GetSystemChannel(netID int) (*Channel, error) {
	net, err := GetNetworkfromNets(netID)
	if err != nil {
		return nil, err
	}
	return &Channel{
		ID: -1,
		Name: "system-channel",
		NetworkID: netID,
		Organizations: []Organization{net.Organizations[0]},
		Orderers: Orders{},
	}, nil
}

func (c *Channel)CreateChannel(ordererURL string) error {
	global.Logger.Info("channel is creating...")

	global.Logger.Info("Update the global variable Nets and insert the new channel into it")
	UpdateNets(*c)

	UpdateSDK(c.NetworkID)

	sdk, err := GetSDKByNetWorkID(c.NetworkID)
	if err != nil {
		return errors.WithMessage(err, "fail to get sdk ")
	}
	channelConfigTxPath := filepath.Join(config.LOCAL_BASE_PATH, fmt.Sprintf("net%d", c.NetworkID), fmt.Sprintf("channel%d.tx", c.ID))

	n, err := GetNetworkfromNets(c.NetworkID)
	if err != nil {
		return err
	}

	global.Logger.Info("Obtaining administrator signature...")
	adminIdentitys, err := n.GetAllAdminSigningIdentities()
	if err != nil {
		return errors.WithMessage(err, "fail to get all SigningIdentities")
	}


	req := resmgmt.SaveChannelRequest{
		ChannelID: fmt.Sprintf("channel%d", c.ID),
		ChannelConfigPath: channelConfigTxPath,
		SigningIdentities: adminIdentitys,
	}

	rcp := sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@org%d.net%d.com", c.Organizations[0].ID, n.ID)), fabsdk.WithOrg(fmt.Sprintf("org%d", c.Organizations[0].ID)))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}

	global.Logger.Info("Submitting to create channel transaction...")
	_, err = rc.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint(ordererURL))


	return err
}

//func (c *Channel) GetBlockByID()
func GetSysChannelConfig(netID int) ([]byte, error) {
	if err := UpdateSDK(netID); err != nil {
		return nil, err
	}
	sdk, err := GetSDKByNetWorkID(netID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk ")
	}

	resmgmtClient, err := resmgmt.New(
		sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@net%d.com", netID)), fabsdk.WithOrg("ordererorg")))
	if err != nil {
		return nil, err
	}

	cfg, err := resmgmtClient.QueryConfigBlockFromOrderer("system-channel", resmgmt.WithOrdererEndpoint(fmt.Sprintf("orderer1.net%d.com", netID)))
	if err != nil {
		return nil, errors.WithMessage(err, "fail to query system-channel config")
	}
	//global.Logger.Info("Obtaining ledgerClient ...")
	//
	//ledgerClient, err := ledger.New(sdk.ChannelContext(
	//	"system-channel",
	//	fabsdk.WithUser(fmt.Sprintf("Admin1@net%d.com", netID)),
	//	fabsdk.WithOrg("ordererorg")))
	//if err != nil {
	//	return nil, err
	//}
	//
	//global.Logger.Info("Query ...")
	//// !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	//peername := fmt.Sprintf("peer1.org1.net%d.com", netID)
	////orderername := fmt.Sprintf("orderer1.net%d.com", netID)
	//cfg, err := ledgerClient.QueryConfigBlock(ledger.WithTargetEndpoints(peername))
	//if err != nil {
	//	return nil, errors.WithMessage(err, "fail to query config")
	//}

	return proto.Marshal(cfg)
}

func (c *Channel) GetChannelConfig() ([]byte, error) {
	if c.ID == -1 {
		return nil, errors.New("please call GetSysChannelConfig")
	}
	fmt.Println(c.Organizations[0].Users[0], c.Organizations[0].ID)
	ledgerClient, err := c.NewLedgerClient(c.Organizations[0].Users[0], fmt.Sprintf("org%d", c.Organizations[0].ID))
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get ledgerClient")
	}

	cfg, err := ledgerClient.QueryConfigBlock(ledger.WithTargetEndpoints(c.Organizations[0].Peers[0].Name))
	if err != nil {
		return nil, errors.WithMessage(err, "fail to query config")
	}

	//fmt.Println(cfg.Header)
	//fmt.Println(cfg.Data)
	//fmt.Println(cfg.Metadata)

	return proto.Marshal(cfg)
}

// Don't use for system-channel
func (c *Channel) GetChannelInfo() (*fab.BlockchainInfoResponse, error) {
	if len(c.Organizations) < 1 {
		return &fab.BlockchainInfoResponse{}, errors.New("No organization in the channel")
	}
	if len(c.Organizations[0].Peers) < 1 {
		return &fab.BlockchainInfoResponse{}, errors.New("No peer in the organization")
	}
	if (len(c.Organizations[0].Users) < 1) {
		return &fab.BlockchainInfoResponse{}, errors.New("No user in the organization")
	}

	lc, err := c.NewLedgerClient(c.Organizations[0].Users[0], fmt.Sprintf("org%d", c.Organizations[0].ID))
	if err != nil {
		return &fab.BlockchainInfoResponse{}, err
	}

	return lc.QueryInfo(ledger.WithTargetEndpoints(c.Organizations[0].Peers[0].Name))
}

// Don't use for system-channel
func (c *Channel) GetBlock(blockID uint64) (*common.Block, error) {
	if len(c.Organizations) < 1 {
		return &common.Block{}, errors.New("No organization in the channel")
	}
	if len(c.Organizations[0].Peers) < 1 {
		return &common.Block{}, errors.New("No peer in the organization")
	}
	if (len(c.Organizations[0].Users) < 1) {
		return &common.Block{}, errors.New("No user in the organization")
	}

	lc, err := c.NewLedgerClient(c.Organizations[0].Users[0], fmt.Sprintf("org%d", c.Organizations[0].ID))
	if err != nil {
		return &common.Block{}, err
	}

	return lc.QueryBlock(blockID, ledger.WithTargetEndpoints(c.Organizations[0].Peers[0].Name))
}

func (c *Channel)getAndStoreConfig() error {
	global.Logger.Info("Obtaining channel configuration ...")
	if c.ID != -1 && (len(c.Organizations) < 1 || len(c.Orderers) < 1) {
		return errors.New("There is no organization in the channel.")
	}

	bt := []byte{}
	var err error
	if c.ID != -1 {
		bt, err = c.GetChannelConfig()
		if err != nil {
			return err
		}
	} else {
		bt, err = GetSysChannelConfig(c.NetworkID)
		if err != nil {
			return err
		}
	}


	global.Logger.Info("Storing channel configuration ...")
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

func (c *Channel)updateConfig(signs []msp.SigningIdentity) error {
	envelopeFile, err := os.Open(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()

	req := resmgmt.SaveChannelRequest{
		ChannelID:         fmt.Sprintf("channel%d", c.ID),
		ChannelConfig:     envelopeFile,
		SigningIdentities: signs,
	}
	resmgmtClient, err := c.NewResmgmtClient(c.Organizations[0].Users[0], fmt.Sprintf("org%d", c.Organizations[0].ID))
	_, err = resmgmtClient.SaveChannel(
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(c.Orderers[0].Name))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}
	return nil
}

// AddOrg uses the existing organization's certificate to update the configuration of the channel
func (c *Channel) AddOrg(orgID int) error {
	global.Logger.Info(fmt.Sprintf("Add org%d to channel%d", orgID, c.ID))
	org := GetBasicOrg(orgID, c.NetworkID)

	global.Logger.Info("Obtaining channel config...")
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	//// 启动ca，获取各种证书
	//if err := org.CreateBasicOrganizationEntity(); err != nil {
	//	return err
	//}

	// generate configtx.yaml
	global.Logger.Info("generate configtx.yaml...")
	configtxFile, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "configtx.yaml"))
	if err != nil {
		return errors.WithMessage(err, "fail to open configtx.yaml")
	}

	_, err = configtxFile.WriteString(org.GetConfigtxFile())
	if err != nil {
		return errors.WithMessage(err, "fail to write configtx.yaml")
	}

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	//cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "addOrg", c.Name, org.MSPID)
	//output, err := cmd.CombinedOutput()
	//global.Logger.Info(string(output))
	global.Logger.Info("generate org_update_in_envelope.pb...")
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrg",
		fmt.Sprintf("channel%d", c.ID),
		org.MSPID,)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	global.Logger.Info("sign for org_update_in_envelope.pb")
	signs := []msp.SigningIdentity{}
	for _, org := range c.Organizations {
		//mspClient, err := org.NewMspClient()
		//if err != nil {
		//	return errors.WithMessage(err, "fail to get mspClient "+org.Name)
		//}
		//adminIdentity, err := mspClient.GetSigningIdentity("Admin")
		//if err != nil {
		//	return errors.WithMessage(err, org.Name+"fail to sign")
		//}
		global.Logger.Info(fmt.Sprintf("Obtaining org%d's adminIdentity", org.ID))
		adminIdentity, err := org.GetAdminSigningIdentity()
		if err != nil {
			global.Logger.Error("fail to get adminIdentity", zap.Error(err))
		}
		signs = append(signs, adminIdentity)
	}


	// update org_update_in_envelope.pb
	global.Logger.Info("Update channel config...")
	if err := c.updateConfig(signs); err != nil {
		return err
	}

	// update channel to Nets
	global.Logger.Info("Update Nets...")
	c.Organizations = append(c.Organizations, *org)
	UpdateNets(*c)

	// TODO
	// return org.CreateNodeEntity()
	return nil
}

func (c *Channel)UpdateAnchors(orgID int) error {
	// 不要让用户自定义了，让所有peer都成为锚节点
	global.Logger.Info("Update anchors...")
	if orgID <= 0 {
		return errors.New(fmt.Sprintf("org ID is incorrect. ID: %d", orgID))
	}

	org := Organization{}
	flag := false
	for _, o := range c.Organizations {
		if orgID == o.ID {
			org = o
			flag = true
			break
		}
	}
	if !flag {
		return errors.New(fmt.Sprintf("The org%d could not be found in channel%d", orgID, c.ID))
	}

	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	// generate anchors.json
	st := `{"mod_policy":"Admins","value":{"anchor_peers":[`
	for _, peer := range org.Peers {
		st += `{"host":"` + NewCaUserFromDomainName(peer.Name).GetURL() + `","port":7051},`
	}
	st += `{"host":"` + "lilingj.github.io" + `","port":7051},`
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
		fmt.Sprintf("channel%d", c.ID),
		org.MSPID)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	for _, org := range c.Organizations {
		//mspClient, err := org.NewMspClient()
		//if err != nil {
		//	return errors.WithMessage(err, "fail to get mspClient "+org.Name)
		//}
		//adminIdentity, err := mspClient.GetSigningIdentity("Admin")
		//if err != nil {
		//	return errors.WithMessage(err, org.Name+"fail to sign")
		//}
		global.Logger.Info(fmt.Sprintf("Obtaining org%d's adminIdentity", org.ID))
		adminIdentity, err := org.GetAdminSigningIdentity()
		if err != nil {
			global.Logger.Error("fail to get adminIdentity", zap.Error(err))
		}
		signs = append(signs, adminIdentity)
	}

	// update org_update_in_envelope.pb
	return c.updateConfig(signs)

}

// AddOrderers
// eg: GetSystemChannel(n.ID).AddOrderers(Order{fmt.Sprintf("orderer%d.net%d.com", len(n.Orderers) + 1, n.ID)})
func (c *Channel)AddOrderers(orderer Order) error {
	global.Logger.Info("Add Orderers ...")
	if c.ID != -1 {
		return errors.New("only for system-channel")
	}

	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}



	// generate ord1.json
	st := `["`
	for _, orderer := range c.Organizations[0].Peers {
		tlscert := NewCaUserFromDomainName(orderer.Name).GetTLSCert(true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) +
			`","host":"` + orderer.Name +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) + `"},`
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
	for _, orderer := range c.Organizations[0].Peers {
		st += `"` + orderer.Name + `",`
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
		fmt.Sprintf("channel%d", c.ID),
		)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	mspClient, err := c.Organizations[0].NewMspClient()
	if err != nil {
		return errors.WithMessage(err, "fail to get mspClient "+c.Organizations[0].Name)
	}
	adminIdentity, err := mspClient.GetSigningIdentity("Admin")
	if err != nil {
		return errors.WithMessage(err, c.Organizations[0].Name+"fail to sign")
	}
	signs = append(signs, adminIdentity)

	// update org_update_in_envelope.pb
	return c.updateConfig(signs)
}

// 渲染一个通道，只包含通道中第一个org
func (c *Channel) RenderConfigtx() error {
	templ := template.Must(template.ParseFiles(path.Join(config.LOCAL_MOUNT_PATH, "channel.yaml.tpl")))

	filename := fmt.Sprintf("/mictract/networks/net%d/configtx.yaml", c.NetworkID)
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	if err := templ.Execute(writer, c.Organizations[0]); err != nil {
		return err
	}

	return nil
}