package model

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-protos-go/common"
	channelclient "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"go.uber.org/zap"
	"mictract/enum"
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
	ID            		int 				`json:"id"`
	Nickname	  		string 				`json:"nickname"`
	NetworkID     		int        			`json:"networkID"`
	Status 		  		string 				`json:"status"`

	OrganizationIDs		ints 				`json:"organization_ids"`
	OrdererIDs          ints				`json:"orderer_ids"`
}

// gorm need
type ints []int
func (arr ints) Value() (driver.Value, error) {
	return json.Marshal(arr)
}
func (arr *ints) Scan(data interface{}) error {
	return json.Unmarshal(data.([]byte), &arr)
}

func NewChannel(netID int, nickname string, orgIDs []int) (*Channel, error) {
	// 1. check
	net, _ := FindNetworkByID(netID)
	if net.Status != enum.StatusRunning {
		return &Channel{}, errors.New("Failed to call NewChannel, network status is abnormal")
	}

	if len(orgIDs) == 0 {
		return &Channel{}, errors.New("Failed to call NewChannel, orgIDs length is at least 1")
	}

	orderers, err := net.GetOrderers()
	if err != nil {
		return &Channel{}, err
	}

	ch := &Channel{
		Nickname: nickname,
		NetworkID: netID,
		Status: enum.StatusStarting,
		OrganizationIDs: orgIDs,
		OrdererIDs: []int{orderers[0].ID},
	}
	if err := global.DB.Create(ch).Error; err != nil {
		return &Channel{}, err
	}
	return ch, nil
}

func GetSystemChannel(netID int) *Channel {
	return &Channel{
		ID: -1,
		NetworkID: netID,
		Status: enum.StatusRunning,
	}
}

func (c *Channel) NewLedgerClient(username, orgname string) (*ledger.Client, error) {
	sdk, err := GetSDKByNetworkID(c.NetworkID)
	if err != nil {
		return &ledger.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}

	ledgerClient, err := ledger.New(sdk.ChannelContext(c.GetName(), fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return &ledger.Client{}, err
	}
	return ledgerClient, nil
}

func (c *Channel) NewResmgmtClient(username, orgname string) (*resmgmt.Client, error) {
	sdk, err := GetSDKByNetworkID(c.NetworkID)
	if err != nil {
		return &resmgmt.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}
	resmgmtClient, err := resmgmt.New(sdk.Context(fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return &resmgmt.Client{}, err
	}
	return resmgmtClient, nil
}

func (c *Channel) NewChannelClient(username, orgname string) (*channelclient.Client, error) {
	sdk, err := GetSDKByNetworkID(c.NetworkID)
	if err != nil {
		return &channelclient.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}
	ccp := sdk.ChannelContext(
		c.GetName(),
		fabsdk.WithUser(username),
		fabsdk.WithOrg(orgname))
	chClient, err := channelclient.New(ccp)
	if err != nil {
		return &channelclient.Client{}, err
	}
	return chClient, nil
}

func (c *Channel) GetName() string {
	if c.ID == -1 {
		return "system-channel"
	}
	return fmt.Sprintf("channel%d", c.ID)
}

func (c *Channel) GetOrderers() ([]CaUser, error) {
	if len(c.OrdererIDs) <= 0 {
		return []CaUser{}, errors.New("no orderer in channel")
	}
	cus := []CaUser{}
	for _, ordererID := range c.OrdererIDs {
		ord, err := FindCaUserByID(ordererID)
		if err != nil {
			return []CaUser{}, err
		}
		cus = append(cus, *ord)
	}
	return cus, nil
}

func (c *Channel) GetOrganizations() ([]Organization, error) {
	if len(c.OrganizationIDs) <= 0 {
		return []Organization{}, errors.New("no organization in channel")
	}
	orgs := []Organization{}
	for _, orgID := range c.OrganizationIDs {
		org, err := FindOrganizationByID(orgID)
		if err != nil {
			return []Organization{}, err
		}
		orgs = append(orgs, *org)
	}
	return orgs, nil
}

func (c *Channel)CreateChannel(ordererURL string) error {
	global.Logger.Info("channel is creating...")

	channelConfigTxPath := filepath.Join(
		config.LOCAL_BASE_PATH,
		fmt.Sprintf("net%d", c.NetworkID),
		fmt.Sprintf("channel%d.tx", c.ID))

	n, err := FindNetworkByID(c.NetworkID)
	if err != nil {
		return err
	}

	// 1. get signs
	global.Logger.Info("Obtaining administrator signature...")
	adminIdentitys, err := n.GetAllAdminSigningIdentities()
	if err != nil {
		return errors.WithMessage(err, "fail to get all SigningIdentities")
	}

	// 2. generate req
	req := resmgmt.SaveChannelRequest{
		ChannelID: c.GetName(),
		ChannelConfigPath: channelConfigTxPath,
		SigningIdentities: adminIdentitys,
	}

	// 3. get rc
	org, err := FindOrganizationByID(c.OrganizationIDs[0])
	if err != nil {
		return err
	}

	adminUser, err := org.GetSystemUser()
	if err != nil {
		return err
	}

	rc, err := c.NewResmgmtClient(adminUser.GetName(), org.GetName())
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}

	// 4. submitting
	global.Logger.Info("Submitting to create channel transaction...")
	_, err = rc.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint(ordererURL))

	return err
}


func GetSysChannelConfig(netID int) ([]byte, error) {
	net, err := FindNetworkByID(netID)
	if err != nil {
		return []byte{}, err
	}
	orderers, err := net.GetOrderers()
	if err != nil {
		return []byte{}, err
	}
	ordOrg, err := net.GetOrdererOrganization()
	if err != nil {
		return []byte{}, err
	}
	ordAdminOrg, err := ordOrg.GetSystemUser()
	if err != nil {
		return []byte{}, err
	}
	resmgmtClient, err := GetSystemChannel(netID).NewResmgmtClient(ordAdminOrg.GetName(), ordOrg.GetName())
	if err != nil {
		return []byte{}, err
	}

	cfg, err := resmgmtClient.QueryConfigBlockFromOrderer(
		"system-channel",
		resmgmt.WithOrdererEndpoint(orderers[0].GetName()))
	if err != nil {
		return []byte{}, errors.WithMessage(err, "fail to query system-channel config")
	}

	return proto.Marshal(cfg)
}

func (c *Channel) GetChannelConfig() ([]byte, error) {
	if c.ID == -1 {
		return nil, errors.New("please call GetSysChannelConfig")
	}

	orgs, err := c.GetOrganizations()
	if err != nil {
		return []byte{}, err
	}

	adminUser, err := orgs[0].GetSystemUser()
	if err != nil {
		return []byte{}, err
	}

	ledgerClient, err := c.NewLedgerClient(adminUser.GetName(), orgs[0].GetName())
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get ledgerClient")
	}

	peers, err := orgs[0].GetPeers()
	if err != nil {
		return []byte{}, err
	}

	cfg, err := ledgerClient.QueryConfigBlock(ledger.WithTargetEndpoints(peers[0].GetName()))
	if err != nil {
		return nil, errors.WithMessage(err, "fail to query config")
	}

	return proto.Marshal(cfg)
}

// Don't use for system-channel
func (c *Channel) GetChannelInfo() (*fab.BlockchainInfoResponse, error) {
	var err error
	var org *Organization
	if len(c.OrganizationIDs) < 1 {
		return &fab.BlockchainInfoResponse{}, errors.New("No organization in the channel")
	}

	for _, orgID := range c.OrganizationIDs {
		org, err = FindOrganizationByID(orgID)
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}
		adminUser, err := org.GetSystemUser()
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}
		lc, err := c.NewLedgerClient(adminUser.GetName(), org.GetName())
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}

		peers, err := org.GetPeers()
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
func (c *Channel) GetBlock(blockID uint64) (*common.Block, error) {
	org, err := FindOrganizationByID(c.OrganizationIDs[0])
	if err != nil {
		global.Logger.Error("", zap.Error(err))
		return &common.Block{}, err
	}
	adminUser, err := org.GetSystemUser()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
		return &common.Block{}, err
	}
	lc, err := c.NewLedgerClient(adminUser.GetName(), org.GetName())
	if err != nil {
		return &common.Block{}, err
	}

	peers, err := org.GetPeers()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
		return &common.Block{}, err
	}
	return lc.QueryBlock(blockID, ledger.WithTargetEndpoints(peers[0].GetName()))
}

func (c *Channel)getAndStoreConfig() error {
	global.Logger.Info("Obtaining channel configuration ...")

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

	orgs, err := c.GetOrganizations()
	if err != nil {
		return err
	}

	adminUser, err := orgs[0].GetSystemUser()
	if err != nil {
		return err
	}

	orderers, err := c.GetOrderers()
	if err != nil {
		return err
	}

	req := resmgmt.SaveChannelRequest{
		ChannelID:         fmt.Sprintf("channel%d", c.ID),
		ChannelConfig:     envelopeFile,
		SigningIdentities: signs,
	}
	resmgmtClient, err := c.NewResmgmtClient(adminUser.GetName(), orgs[0].GetName())
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
func (c *Channel) AddOrg(orgID int) error {
	global.Logger.Info(fmt.Sprintf("Add org%d to channel%d", orgID, c.ID))
	org, err := FindOrganizationByID(orgID)
	if err != nil {
		return err
	}

	global.Logger.Info("Obtaining channel config...")
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

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
		c.GetName(),
		org.GetMSPID(),)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	global.Logger.Info("sign for org_update_in_envelope.pb")
	signs := []msp.SigningIdentity{}

	orgs, err := c.GetOrganizations()
	if err != nil {
		return err
	}

	for _, org := range orgs {
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

	return UpdateOrgIDs(c.ID, orgID)
}

func (c *Channel)UpdateAnchors(orgID int) error {
	// 不要让用户自定义了，让所有peer都成为锚节点
	global.Logger.Info("Update anchors...")
	if orgID <= 0 {
		return errors.New(fmt.Sprintf("org ID is incorrect. ID: %d", orgID))
	}

	org, err := FindOrganizationByID(orgID)
	if err != nil {
		return err
	}

	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	// generate anchors.json
	peers, err := org.GetPeers()
	if err != nil {
		return err
	}

	st := `{"mod_policy":"Admins","value":{"anchor_peers":[`
	for _, peer := range peers {
		st += `{"host":"` + peer.GetURL() + `","port":7051},`
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
		c.GetName(),
		org.GetMSPID())
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	orgs, err := c.GetOrganizations()
	if err != nil {
		return err
	}

	signs := []msp.SigningIdentity{}
	for _, org := range orgs {
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
func (c *Channel)AddOrderers(orderer CaUser) error {
	global.Logger.Info("Add Orderers ...")
	if c.ID != -1 {
		return errors.New("only for system-channel")
	}

	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	// generate ord1.json
	orderers, err := c.GetOrderers()
	if err != nil {
		return err
	}

	st := `["`
	for _, orderer := range orderers {
		tlscert := NewCaUserFromDomainName(orderer.GetURL()).GetTLSCert(true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) +
			`","host":"` + orderer.GetURL() +
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
		fmt.Sprintf("channel%d", c.ID),
		)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	orgs, err := c.GetOrganizations()
	if err != nil {
		return err
	}

	signs := []msp.SigningIdentity{}
	mspClient, err := orgs[0].NewMspClient()
	if err != nil {
		return errors.WithMessage(err, "fail to get mspClient ")
	}
	adminIdentity, err := mspClient.GetSigningIdentity("Admin")
	if err != nil {
		return errors.WithMessage(err, "fail to sign")
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

	orgs, err := c.GetOrganizations()
	if err != nil {
		return err
	}

	if err := templ.Execute(writer, orgs[0]); err != nil {
		return err
	}

	return nil
}

func FindChannelByID(chID int) (*Channel, error) {
	var chs []Channel
	if err := global.DB.Where("id = ?", chID).Find(&chs).Error; err != nil {
		return &Channel{}, err
	}
	return &chs[0], nil
}

func UpdateOrgIDs(chID, orgID int) error {
	// 加个互斥锁
	global.ChannelLock.Lock()
	defer global.ChannelLock.Unlock()

	ch, err := FindChannelByID(chID)
	if err != nil {
		return err
	}

	ch.OrganizationIDs = append(ch.OrganizationIDs, orgID)

	return global.DB.Model(ch).Update("organization_ids", ch.OrganizationIDs).Error
}

func (c *Channel) UpdateStatus(status string) error {
	return global.DB.Model(&Channel{}).Where("id = ?", c.ID).Update("status", status).Error
}