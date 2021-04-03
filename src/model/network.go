package model

import (
	"encoding/base64"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	mConfig "mictract/config"
	"mictract/enum"
	"mictract/global"
	"mictract/model/kubernetes"
	"mictract/model/request"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"

	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type Network struct {
	ID        	int 		`json:"id" gorm:"primarykey"`
	Nickname	string 		`json:"nickname"`
	CreatedAt 	time.Time
	Status 		string 		`json:"status"`

	Consensus  	string 		`json:"consensus" binding:"required"`
	TlsEnabled 	bool   		`json:"tlsEnabled"`
}

func NewNetwork(nickname, consensus string) (*Network, error) {
	// 1. check
	if consensus != "solo" && consensus != "etcdraft" {
		return &Network{}, errors.New("only supports solo and etcdraft")
	}

	// 2. new
	net := &Network{
		Nickname: nickname,
		CreatedAt: time.Now(),
		Status: "starting",
		Consensus: consensus,
	}

	// 3. insert into db
	if err := global.DB.Create(net).Error; err != nil {
		return &Network{}, errors.WithMessage(err, "Unable to insert network")
	}
	return net, nil
}

func (n *Network) GetName() string {
	return fmt.Sprintf("net%d", n.ID)
}

func (n *Network) RemoveAllFile() {
	if err := os.RemoveAll(filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", n.ID))); err != nil {
		global.Logger.Error("fail to remove all file", zap.Error(err))
	}
}


func FindNetworkByID(id int) (*Network, error) {
	var nets []Network
	if err := global.DB.Where("id = ?", id).Find(&nets).Error; err != nil {
		global.Logger.Error(err.Error())
		return &Network{}, err
	}
	if len(nets) == 0 {
		return &Network{}, errors.New("no such network")
	}
	return &nets[0], nil
}

func DeleteNetworkByID(id int) error {
	var net *Network
	var orgs []Organization
	var ccs []Chaincode
	var sdk *fabsdk.FabricSDK
	var err error
	var ok bool

	net, err = FindNetworkByID(id)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	// 2 remove organizations entity
	if err = global.DB.Where("network_id = ?", id).Find(&orgs).Error; err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, org := range orgs {
		org.RemoveAllEntity()
	}

	// 3 remove chaincode entity
	if ccs, err = net.GetChaincodes(); err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, cc := range ccs {
		cc.RemoveEntity()
	}

	// 4. delete global.SDKs
	sdk, ok = global.SDKs[id]
	if ok {
		sdk.Close()
		delete(global.SDKs, id)
	}

	// 5. delete from networks where id = id
	if err := global.DB.Where("id = ?", id).Delete(&Network{}).Error; err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	// 6.1 delete from organizations where network_id = id
	// 6.2 delete from ca_users where network_id = id
	// 6.3 delete from chaincode where network_id = id
	// 6.4 delete from channels where network_id = id
	for _, itf := range []interface{}{
		&Organization{},
		&CaUser{},
		&Chaincode{},
		&Channel{}} {
		if err := global.DB.Where("network_id = ?", id).Delete(itf).Error; err != nil {
			global.Logger.Error("", zap.Error(err))
		}
	}

	// 6. TODO: RemoveAllFile

	return nil
}

func FindAllNetworks() ([]Network, error){
	nets := []Network{}
	if err := global.DB.Find(&nets).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all nets")
	}
	return nets, nil
}

func UpdateNetworkStatusByID(id int, status string) error {
	return global.DB.Model(&Network{}).Where("id = ?", id).Update("status", status).Error
}

// Deploy method is just creating a basic network containing only 1 peer and 1 orderer,
//	and then join the rest of peers and orderers.
// The basic network is built to make `configtx.yaml` file simple enough to create the genesis block.
func Deploy(addNetReq request.AddNetworkReq) (*Network, error) {
	global.Logger.Info("Deploying network...")

	tools := kubernetes.Tools{}

	// 1. insert new network object to db
	net, err := NewNetwork(addNetReq.Nickname, addNetReq.Consensus)
	if err != nil {
		return net, err
	}

	fmt.Printf("%+v\n", net)

	ordererOrg, err := NewOrdererOrganization(net.ID, "ordererorg")
	if err != nil {
		return net, err
	}
	// 启动ca节点并获取基础组织的证书
	if err := ordererOrg.CreateBasicOrganizationEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg", zap.Error(err))
	}

	//org1, err := NewOrganization(net.ID, addNetReq.OrgNicknames[0])
	//if err != nil {
	//	return &Network{}, err
	//}
	//if err := org1.CreateBasicOrganizationEntity(); err != nil {
	//	global.Logger.Error("fail to start org1", zap.Error(err))
	//}

	// configtx.yaml should be placed in `networks/netX/configtx.yaml`
	global.Logger.Info("Render configtx.yaml")
	orderers, err := net.GetOrderers()
	if err != nil {
		return net, err
	}

	if err = orderers[0].RenderConfigtx(); err != nil {
		return net, err
	}

	// generate the genesis block
	global.Logger.Info("generate the genesis block...")
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", net.ID),
		"-profile", "Genesis",
		"-channelID", "system-channel",
		"-outputBlock", fmt.Sprintf("/mictract/networks/net%d/genesis.block", net.ID),
	)
	if err != nil {
		return net, err
	}

	// 启动组织的剩余节点，一个peer或者一个orderer
	if err := ordererOrg.CreateNodeEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg's node", zap.Error(err))
	}
	//if err := org1.CreateNodeEntity(); err != nil {
	//	global.Logger.Error("fail to start org1's node", zap.Error(err))
	//}

	return net, nil
}

// only for OrdererOrg
func (orderer *CaUser) RenderConfigtx() error {
	templ := template.Must(template.ParseFiles(path.Join(mConfig.LOCAL_MOUNT_PATH, "configtx.yaml.tpl")))

	filename := fmt.Sprintf("/mictract/networks/net%d/configtx.yaml", orderer.NetworkID)
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	if err := templ.Execute(writer, orderer); err != nil {
		return err
	}

	return nil
}

func (n *Network)GetAllAdminSigningIdentities() ([]msp.SigningIdentity, error) {
	orgs, err := n.GetOrganizations()
	if err != nil {
		return []msp.SigningIdentity{}, err
	}

	signs := []msp.SigningIdentity{}
	for _, org := range orgs {
		mspClient, err := org.NewMspClient()
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d mspClient", org.ID), zap.Error(err))
		}
		adminUser, err := org.GetSystemUser()
		if err != nil {
			global.Logger.Error("fail to get system-user", zap.Error(err))
		}

		// 公私钥写入sdk配置文件中，可直接读取，不需要enroll
		//if err := mspClient.Enroll(username, mspclient.WithSecret(password)); err != nil {
		//	global.Logger.Error("fail to enroll user " + username, zap.Error(err))
		//}
		global.Logger.Info(fmt.Sprintf("Obtaining %s publ and priv", adminUser.GetName()))

		sign, err := mspClient.GetSigningIdentity(adminUser.GetName())
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d AdminSigningIdentity", org.ID), zap.Error(err))
		}
		signs = append(signs, sign)
	}

	return signs, nil
}

// AddOrderers
func (net *Network)AddOrderersToSystemChannel() error {
	global.Logger.Info("Add Orderer to system-channel ...")

	if net.Consensus == "solo" {
		return errors.New("Does not support networks that use the solo protocol")
	}

	c := GetSystemChannel(net.ID)
	orderers, err := c.GetOrderers()
	if err != nil {
		return err
	}
	ordOrg, err := net.GetOrdererOrganization()
	if err != nil {
		return err
	}
	adminUser, err := ordOrg.GetSystemUser()
	if err != nil {
		return err
	}

	// generate config_block.pb
	global.Logger.Info("Get and Store system-channel config ...")
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	mspClient, err := ordOrg.NewMspClient()
	if err != nil {
		return err
	}
	// regiester new orderer
	global.Logger.Info("Regiester new orderer")
	user, err := NewOrdererCaUser(ordOrg.ID, net.ID, "orderer1")
	if err != nil {
		return err
	}
	if err := user.Register(mspClient); err != nil {
		return err
	}

	// enroll new orderer
	global.Logger.Info("Enroll new orderer")
	if err := user.Enroll(mspClient, true); err != nil {
		return err
	}
	if err := user.Enroll(mspClient, false); err != nil {
		return err
	}

	global.Logger.Info("orderer starts creating")
	if err := kubernetes.NewOrderer(net.ID, user.ID).AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("orderer has been created synchronously")

	// generate ord1.json
	st := `[`
	for _, orderer := range orderers {
		tlscert := orderer.GetTLSCert(true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) +
			`","host":"` + orderer.GetURL() +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) + `"},`
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

	// generate ord2.json
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

	// call addorg.sh to generate org_update_in_envelope.pb
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrderers",
		"system-channel",
	)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	adminIdentity, err := mspClient.GetSigningIdentity(adminUser.GetName())
	if err != nil {
		return errors.WithMessage(err, "ordererAdmin fail to sign")
	}
	signs = append(signs, adminIdentity)

	// update org_update_in_envelope.pb
	global.Logger.Info("update org_update_in_envelope.pb...")
	envelopeFile, err := os.Open(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()


	resmgmtClient, err := c.NewResmgmtClient(adminUser.GetName(), ordOrg.GetName())
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

func (n *Network) AddOrgToConsortium(orgID int) error {
	global.Logger.Info(fmt.Sprintf("Add org%d to Consortium(Write to system-channel)...", orgID))

	orderers, err := n.GetOrderers()
	if err != nil {
		return err
	}
	org, err := FindOrganizationByID(orgID)
	if err != nil {
		return err
	}
	ordOrg, err := n.GetOrdererOrganization()
	if err != nil {
		return err
	}
	ordAdminUser, err := ordOrg.GetSystemUser()
	if err!= nil {
		return err
	}

	global.Logger.Info("Obtaining channel config...")
	sysch := GetSystemChannel(n.ID)
	if err :=sysch.getAndStoreConfig(); err != nil {
		return err
	}

	global.Logger.Info("generate configtx.yaml...")
	configtxFile, err := os.Create(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "configtx.yaml"))
	if err != nil {
		return errors.WithMessage(err, "fail to open configtx.yaml")
	}
	_, err = configtxFile.WriteString(org.GetConfigtxFile())
	if err != nil {
		return errors.WithMessage(err, "fail to write configtx.yaml")
	}

	global.Logger.Info("generate org_update_in_envelope.pb...")
	tools := kubernetes.Tools{}
	_, _, err = tools.ExecCommand(
		filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"),
		"addOrgToConsortium",
		"system-channel",
		org.GetMSPID())
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}


	mspClient, err := ordOrg.NewMspClient()
	if err != nil {
		return err
	}

	signs := []msp.SigningIdentity{}
	adminIdentity, err := mspClient.GetSigningIdentity(ordAdminUser.GetName())
	if err != nil {
		return errors.WithMessage(err, "ordererAdmin fail to sign")
	}
	signs = append(signs, adminIdentity)

	global.Logger.Info("update org_update_in_envelope.pb...")
	envelopeFile, err := os.Open(filepath.Join(mConfig.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}
	defer envelopeFile.Close()

	resmgmtClient, err := GetSystemChannel(n.ID).NewResmgmtClient(ordAdminUser.GetName(), ordOrg.GetName())
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

// AddOrg creates an organizational entity
func (n *Network) AddOrg(nickname string) (*Organization, error) {
	org, err := NewOrganization(n.ID, nickname)
	if err != nil {
		return &Organization{}, err
	}
	if err := org.CreateBasicOrganizationEntity(); err != nil {
		org.UpdateStatus(enum.StatusError)
		return &Organization{}, err
	}
	if err := org.CreateNodeEntity(); err != nil {
		org.UpdateStatus(enum.StatusError)
		return &Organization{}, err
	}
	if err := n.AddOrgToConsortium(org.ID); err != nil {
		org.UpdateStatus(enum.StatusError)
		return &Organization{}, err
	}

	return org, nil
}

func (n *Network) AddChannel(orgIDs []int, nickname string) (*Channel, error) {
	ch, err := NewChannel(n.ID, nickname, orgIDs)
	if err != nil {
		return ch, err
	}
	orgs, err := ch.GetOrganizations()
	if err != nil {
		return ch, err
	}
	orderers, err := ch.GetOrderers()
	if err != nil {
		return ch, err
	}

	if err := ch.RenderConfigtx(); err != nil {
		return ch, errors.WithMessage(err, "fail to render configtx.yaml")
	}

	tools := kubernetes.Tools{}
	global.Logger.Info("generate a default channel")
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "NewChannel",
		"-channelID", ch.GetName(),
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel%d.tx", n.ID, ch.ID),
	)
	if err != nil {
		return ch, err
	}

	if err := ch.CreateChannel(orderers[0].GetName()); err != nil {
		return ch, err
	}

	global.Logger.Info("First peer join channel...")
	peers, err := orgs[0].GetPeers()
	global.Logger.Info(fmt.Sprintf("%+v\n", peers))
	for _, peer := range peers {
		if err := peer.JoinChannel(ch.ID, orderers[0].GetName()); err != nil {
			global.Logger.Error("", zap.Error(err))
		}
	}

	for i := 1; i < len(orgIDs); i++ {
		global.Logger.Info(fmt.Sprintf("add org%d to channel%d", orgIDs[i], ch.ID))
		if err := ch.AddOrg(orgIDs[i]); err != nil {
			return ch, err
		}
	}
	// bug: orderer处理更新配置文件需要时间, 加入通道操作在orderer还每处理完通道配置更新交易就提交上去，
	//      会导致身份识别错误。注意这个玄学问题
	time.Sleep(5 * time.Second)

	global.Logger.Info("Rest peer join channel...")
	for i := 1; i < len(orgs); i++ {
		org := orgs[i]
		peers, err := org.GetPeers()
		if err != nil {
			global.Logger.Error("", zap.Error(err))
			continue
		}
		for _, peer := range peers {
			if err := peer.JoinChannel(
				ch.ID,
				orderers[0].GetName()); err != nil {
				global.Logger.Error(fmt.Sprintf("%s fail to join channel%d", peer.GetName(), ch.ID), zap.Error(err))
			}
		}
	}

	return ch, nil
}

func (n *Network) GetOrganizations() ([]Organization, error) {
	orgs := []Organization{}
	if err := global.DB.Where("network_id = ? and is_orderer_org = ?", n.ID, false).Find(&orgs).Error; err != nil {
		return []Organization{}, err
	}
	return orgs, nil
}

func (n *Network) GetOrdererOrganization() (*Organization, error) {
	orgs := []Organization{}
	if err := global.DB.Where("network_id = ? and is_orderer_org = ?", n.ID, true).Find(&orgs).Error; err != nil {
		return &Organization{}, err
	}
	return &orgs[0], nil
}

func (n *Network) GetOrderers() ([]CaUser, error) {
	cus := []CaUser{}
	if err := global.DB.Where("network_id = ? and type = ?", n.ID, "orderer").Find(&cus).Error; err != nil {
		return []CaUser{}, err
	}
	if len(cus) == 0 {
		return []CaUser{}, errors.New("no orderer in network")
	}
	return cus, nil
}

func (n *Network) GetChannels() ([]Channel, error) {
	chs := []Channel{}
	if err := global.DB.Where("network_id = ?", n.ID).Find(&chs).Error; err != nil {
		return []Channel{}, err
	}
	return chs, nil
}

func (n *Network) GetChaincodes() ([]Chaincode, error) {
	ccs := []Chaincode{}
	if err := global.DB.Where("network_id = ?", n.ID).Find(&ccs).Error; err != nil {
		return []Chaincode{}, err
	}
	return ccs, nil
}

func (n *Network) UpdateStatus(status string) error {
	return global.DB.Model(&Network{}).Where("id = ?", n.ID).Update("status", status).Error
}
