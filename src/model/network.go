package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"
	mConfig "mictract/config"
	"mictract/global"
	"mictract/model/kubernetes"
	"mictract/model/request"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"text/template"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"gopkg.in/yaml.v3"
)

type Network struct {
	ID        int `json:"id" gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	
	Status string `json:"status"`

	Name          string        `json:"name" binding:"required"`
	Orders        Orders        `json:"orders" binding:"required" gorm:"type:text"`
	Organizations Organizations `json:"organizations" binding:"required" gorm:"type:text"`
	Channels      Channels      `json:"channels" gorm:"type:text"`

	Consensus  string `json:"consensus" binding:"required"`
	TlsEnabled bool   `json:"tlsEnabled"`
}

var (
	// just demo
	// one orderer one org one peer
	networks = []Network{
		{
			ID: 1,
			Name: "net1",
			Orders: []Order{},
			Channels: []Channel{},
			Organizations: []Organization{
				{
					ID: -1,
					NetworkID: 1,
					Name: "ordererorg",
					Peers: []Peer{},
					Users: []string{},
				},
				{
					ID: 1,
					NetworkID: 1,
					Name: "org1",
					Peers: []Peer{},
					Users: []string{},
				},
			},
			Consensus:  "solo",
			TlsEnabled: true,
		},
	}
)

func FindNetworks(pageInfo request.PageInfo) ([]Network, error) {
	// TODO
	// find all networks in the `/networks` directory
	start := pageInfo.PageSize * (pageInfo.Page - 1)
	end := pageInfo.PageSize * pageInfo.Page
	if end > len(global.Nets) {
		end = len(global.Nets)
	}
	networks := []Network{}
	for _, v := range global.Nets {
		networks = append(networks, v.(Network))
	}
	return networks[start:end], nil
}

func FindNetworkByID(id int) (*Network, error) {
	return GetNetworkfromNets(id)
}



func DeleteNetworkByID(id int) error {
	if err := global.DB.Where("id = ?", id).Delete(&Network{}).Error; err != nil {
		return errors.WithMessage(err, "Unable to delete network")
	}
	n, _ := GetNetworkfromNets(id)
	n.RemoveAllEntity()
	// n.RemoveAllFile()
	delete(global.Nets, fmt.Sprintf("net%d", id))
	delete(global.SDKs, fmt.Sprintf("net%d", id))
	return nil
}

func QueryAllNetwork() ([]Network, error){
	nets := []Network{}
	if err := global.DB.Find(&nets).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all nets")
	}
	return nets, nil
}

//
func (n *Network)Insert() error {
	if err := global.DB.Create(n).Error; err != nil {
		return errors.WithMessage(err, "Unable to insert network")
	}
	return nil
}

func UpsertAllNets()  {
	for _, net := range global.Nets {
		if err := global.DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
		}).Create(net.(Network)).Error; err != nil {
			global.Logger.Error(fmt.Sprintf("fail to upsert net%d: ",  net.(Network).ID), zap.Error(err))
		}
	}
}

// update to db !!!!
func (n *Network)Update() error {
	nets, err := QueryAllNetwork()
	if err != nil {
		return errors.WithMessage(err, "No such network found")
	}

	isExist := false
	for _, net := range nets {
		if n.ID == net.ID {
			isExist = true
			break
		}
	}
	if !isExist {
		return errors.New("No such network found")
	}

	n.UpdatedAt  = time.Now()
	if err := global.DB.Model(&Network{}).Where("id = ?", n.ID).Updates(n).Error; err != nil {
		return errors.WithMessage(err, "Fail to update")
	}

	return nil
}

// Get Network from Nets
func GetNetworkfromNets(networkID int) (*Network, error) {
	n, ok:= global.Nets[fmt.Sprintf("net%d", networkID)]
	if !ok {
		return nil, errors.New(fmt.Sprintf("The net%d is not deployed, and no records can be queried in Nets", networkID))
	}
	net := n.(Network)
	return &net, nil
}

// Get basic Network for deploy
// eg: GetBasicNetWork().Deploy()
func GetBasicNetwork(consensus string) *Network {
	netID := 0
	for k, _ := range global.Nets {
		curID, _ := strconv.Atoi(k[3:])
		if netID < curID {
			netID = curID
		}
	}
	netID++
	return &Network{
		ID: netID,
		Name: fmt.Sprintf("net%d", netID),
		Orders: []Order{},
		Organizations: []Organization{},
		Channels: []Channel{},
		Consensus: consensus,
		TlsEnabled: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status: "starting",
	}
}

func (n *Network)InitNetsForThisNet() {
	global.Nets[fmt.Sprintf("net%d", n.ID)] = *n
	_, _ = global.Nets[fmt.Sprintf("net%d", n.ID)].(Network)
}

// Deploy method is just creating a basic network containing only 1 peer and 1 orderer,
//	and then join the rest of peers and orderers.
// The basic network is built to make `configtx.yaml` file simple enough to create the genesis block.
func (n *Network) Deploy() (err error) {
	global.Logger.Info("Deploying network...")

	global.Logger.Info("Initialize the network, update the global variable Nets")
	n.InitNetsForThisNet()
	n.Insert()

	ordererOrg := GetBasicOrg(-1, n.ID)
	org1 := GetBasicOrg(1, n.ID)
	channel := Channel{
		ID: 1,
		Name: "channel1",
		NetworkID: n.ID,
		Organizations: Organizations{
			//*ordererOrg,
			*org1,
		},
		Orderers: []Order {
			{fmt.Sprintf("orderer1.net%d.com", n.ID)},
		},
		Status: "starting",
	}

	// create tools resources
	//global.Logger.Info("Start the tools node")
	tools := kubernetes.Tools{}
	//tools.Create()

	// 启动ca节点并获取基础组织的证书
	if err := ordererOrg.CreateBasicOrganizationEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg", zap.Error(err))
	}
	if err := org1.CreateBasicOrganizationEntity(); err != nil {
		global.Logger.Error("fail to start org1", zap.Error(err))
	}

	// configtx.yaml should be placed in `networks/netX/configtx.yaml`
	global.Logger.Info("Render configtx.yaml")
	if err = n.RenderConfigtx(); err != nil {
		return err
	}
	// generate the genesis block
	global.Logger.Info("generate the genesis block...")
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "Genesis",
		"-channelID", "system-channel",
		"-outputBlock", fmt.Sprintf("/mictract/networks/net%d/genesis.block", n.ID),
	)
	if err != nil {
		return err
	}

	// 启动组织的剩余节点，一个peer或者一个orderer
	if err := ordererOrg.CreateNodeEntity(); err != nil {
		global.Logger.Error("fail to start ordererOrg's node", zap.Error(err))
	}
	if err := org1.CreateNodeEntity(); err != nil {
		global.Logger.Error("fail to start org1's node", zap.Error(err))
	}


	// generate a default channel
	global.Logger.Info("generate a default channel")
	if _, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "DefaultChannel",
		"-channelID", "channel1",
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel1.tx", n.ID),
	); err != nil {
		return err
	}

	// Create first Channel channl1
	if err := channel.CreateChannel(fmt.Sprintf("orderer1.net%d.com", n.ID)); err != nil {
		return errors.WithMessage(err, "fail to create channel")
	}
	ch, _ := GetChannelFromNets(channel.ID, n.ID)
	ch.Status = "success"
	UpdateNets(*ch)

	// TODO: join peers into the first channel
	n, err = GetNetworkfromNets(n.ID)
	if err != nil {
		global.Logger.Error("Unable to get the latest network", zap.Error(err))
	}
	if err := n.Organizations[1].Peers[0].JoinChannel("channel1", fmt.Sprintf("orderer1.net%d.com", n.ID)); err != nil {
		return errors.WithMessage(err, "fail to join channel")
	}

	o, _ := GetOrgFromNets(org1.ID, n.ID)
	o.Status = "success"
	UpdateNets(o)

	return nil
}

func (n *Network) RemoveAllEntity() {
	for _, org := range n.Organizations {
		org.RemoveAllEntity()
	}
}

func (n *Network) RemoveAllFile() {
	if err := os.RemoveAll(filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", n.ID))); err != nil {
		global.Logger.Error("fail to remove all file", zap.Error(err))
	}
}

func (n *Network) RenderConfigtx() error {
	templ := template.Must(template.ParseFiles(path.Join(mConfig.LOCAL_MOUNT_PATH, "configtx.yaml.tpl")))

	filename := fmt.Sprintf("/mictract/networks/net%d/configtx.yaml", n.ID)
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	if err := templ.Execute(writer, n); err != nil {
		return err
	}

	return nil
}

// update sdk by globals.Nets
func UpdateSDK(networkID int) error {
	net := global.Nets[fmt.Sprintf("net%d", networkID)].(Network)
	sdkconfig, err := yaml.Marshal(NewSDKConfig(&net))
	if err != nil {
		return err
	}

	// global.Logger.Info(string(sdkconfig))
	// for debug
	f, _ := os.Create(filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", networkID), "sdk-config.yaml"))
	_, _ = f.WriteString(string(sdkconfig))
	_ = f.Close()

	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		return err
	}
	if _, ok := global.SDKs[fmt.Sprintf("net%d", networkID)]; ok {
		global.SDKs[fmt.Sprintf("net%d", networkID)].Close()
	}
	global.SDKs[fmt.Sprintf("net%d", networkID)] = sdk
	return nil
}

//
func (n *Network) GetSDK() (*fabsdk.FabricSDK, error) {
	if _, ok := global.SDKs[fmt.Sprintf("net%d", n.ID)]; !ok {
		if err := UpdateSDK(n.ID); err != nil {
			return nil, err
		}
	}
	return global.SDKs[fmt.Sprintf("net%d", n.ID)], nil
}

func GetSDKByNetWorkID(id int) (*fabsdk.FabricSDK, error) {
	global.Logger.Info("current SDK:")
	for k, _ := range global.SDKs {
		global.Logger.Info("|" + k)
	}
	n := Network{Name: fmt.Sprintf("net%d", id)}
	global.Logger.Info("get sdk " + n.Name)
	//return n.GetSDK()
	sdk, ok := global.SDKs[n.Name]
	if !ok {
		return nil, errors.New("please update SDK")
	}
	return sdk, nil
}

func UpdateNets(v interface{}) {
	global.Logger.Info("UpdateNets ing...")
	global.Logger.Info("!!!!!!Null pointer exception may occur!!!!!!!")
	switch non := v.(type) {
	case Network:
		global.Nets[fmt.Sprintf("net%d", non.ID)] = non
		non.Show()
		if err := non.Update(); err != nil {
			global.Logger.Error("fail to update net to mysql", zap.Error(err))
		}
		//if err := non.UpdateSDK(); err != nil {
		//	global.Logger.Error("fail to update sdk", zap.Error(err))
		//}
	case Organization:
		org := non
		n := global.Nets[fmt.Sprintf("net%d", org.NetworkID)].(Network)
		if org.ID == -1 {
			if len(n.Organizations) == 0 {
				n.Organizations = append(n.Organizations, org)
			} else {
				n.Organizations[0] = org
			}
			// update orderer
			for _, orderer := range org.Peers {
				n.Orders = append(n.Orders, Order{
					orderer.Name,
				})
			}
		} else {
			if len(n.Organizations) > org.ID {
				n.Organizations[org.ID] = org
			} else if len(n.Organizations) == org.ID {
				n.Organizations = append(n.Organizations, org)
			} else {
				panic("The organization ID should be incremented! The ID of the newly added organization must be equal to the largest ID of the existing organization plus one")
			}
		}

		UpdateNets(n)
	case *CaUser:
		cu := non
		n := global.Nets[fmt.Sprintf("net%d", cu.NetworkID)].(Network)
		if cu.Type == "peer" {
			if cu.UserID - 1 == len(n.Organizations[cu.OrganizationID].Peers){
				n.Organizations[cu.OrganizationID].Peers = append(n.Organizations[cu.OrganizationID].Peers, Peer{Name: cu.GetUsername()})
			} else {
				global.Logger.Info("The peer node already exists")
			}

			//peers := n.Organizations[cu.OrganizationID].Peers
			//peers = append(peers, Peer{Name: cu.GetUsername()})
		} else{
			if cu.IsInOrdererOrg() {
				if cu.Type == "admin" || cu.Type == "user"{
					n.Organizations[0].Users = append(n.Organizations[0].Users, cu.GetUsername())
				} else {
					n.Orders = append(n.Orders, Order{Name: cu.GetUsername()})
					n.Organizations[0].Peers = append(n.Organizations[0].Peers, Peer{Name: cu.GetUsername()})
				}
			} else {
				n.Organizations[cu.OrganizationID].Users = append(n.Organizations[cu.OrganizationID].Users, cu.GetUsername())
			}
		}
		// jump
		UpdateNets(n)
	case Channel:
		c := non
		n := global.Nets[fmt.Sprintf("net%d", c.NetworkID)].(Network)
		flag := false
		for i, channel := range n.Channels {
			if channel.ID == c.ID {
				n.Channels[i] = c
				flag = true
				break
			}
		}
		if !flag {
			n.Channels = append(n.Channels, c)
		}
		//jump
		UpdateNets(n)
	default:
		global.Logger.Error("UpdateNets only support type(*CaUser, Network, Channel, Organization)")
	}
}

func (n *Network)GetAllAdminSigningIdentities() ([]msp.SigningIdentity, error) {
	signs := []msp.SigningIdentity{}
	// peer admin (n.Orgnaizations include ordererorg
	for _, org := range n.Organizations {
		username := fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, n.ID)
		if org.ID == -1 {
			username = fmt.Sprintf("Admin1@net%d.com", n.ID)
		}
		// password := "admin1pw"

		mspClient, err := org.NewMspClient()
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d mspClient", org.ID), zap.Error(err))
		}

		// 公私钥写入sdk配置文件中，可直接读取，不需要enroll
		//if err := mspClient.Enroll(username, mspclient.WithSecret(password)); err != nil {
		//	global.Logger.Error("fail to enroll user " + username, zap.Error(err))
		//}
		global.Logger.Info(fmt.Sprintf("Obtaining %s publ and priv", username))
		sign, err := mspClient.GetSigningIdentity(username)
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d AdminSigningIdentity", org.ID), zap.Error(err))
		}
		signs = append(signs, sign)
	}

	return signs, nil
}


func (n *Network)Show() {
	out, err := json.Marshal(n)
	if err != nil {
		global.Logger.Error("fail to show network", zap.Error(err))
	}
	fmt.Println(string(out))
}

// AddOrderers
// eg: GetSystemChannel(n.ID).AddOrderers()
func (net *Network)AddOrderersToSystemChannel() error {
	global.Logger.Info("Add Orderer to system-channel ...")

	n, err := GetNetworkfromNets(net.ID)
	if err != nil {
		return err
	}

	if n.Consensus == "solo" {
		return errors.New("Does not support networks that use the solo protocol")
	}

	c, err := GetSystemChannel(n.ID)
	if err != nil {
		return err
	}

	// generate config_block.pb
	global.Logger.Info("Get and Store system-channel config ...")
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	newOrdererID := len(n.Orders) + 1
	//newOrderer := Order{
	//	Name: fmt.Sprintf("orderer%d.net%d.com", newOrdererID, n.ID),
	//}
	//n.Orders = append(n.Orders, newOrderer)
	//UpdateNets(*n)


	UpdateSDK(n.ID)
	sdk, err := GetSDKByNetWorkID(n.ID)
	if err != nil {
		return err
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(fmt.Sprintf("ca.net%d.com", n.ID)), mspclient.WithOrg("ordererorg"))
	if err != nil {
		return err
	}
	// regiester new orderer
	global.Logger.Info("Regiester new orderer")
	user := NewOrdererCaUser(newOrdererID, n.ID, fmt.Sprintf("orderer%dpw", newOrdererID))
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
	if err := kubernetes.NewOrderer(n.ID, newOrdererID).AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("orderer has been created synchronously")

	n, _ = GetNetworkfromNets(n.ID)
	// generate ord1.json
	st := `[`
	for _, orderer := range n.Orders {
		user := NewCaUserFromDomainName(orderer.Name)
		tlscert := user.GetTLSCert(true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) +
			`","host":"` + user.GetURL() +
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
	for _, orderer := range n.Orders {
		st += `"` + orderer.Name + `",`
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
	// TODO
	//cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "addOrderers", c.Name)
	//output, err := cmd.CombinedOutput()
	//global.Logger.Info(string(output))
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
	adminIdentity, err := mspClient.GetSigningIdentity(fmt.Sprintf("Admin1@net%d.com", n.ID))
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

	resmgmtClient, err := resmgmt.New(
		sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@net%d.com", c.NetworkID)), fabsdk.WithOrg("ordererorg")))
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
		resmgmt.WithOrdererEndpoint(fmt.Sprintf("orderer1.net%d.com", c.NetworkID)))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}
	return nil
}

func (n *Network) AddOrgToConsortium(orgID int) error {
	global.Logger.Info(fmt.Sprintf("Add org%d to Consortium(Write to system-channel)...", orgID))
	net, err := GetNetworkfromNets(n.ID)
	if err != nil {
		return err
	}
	org, err := GetOrgFromNets(orgID, net.ID)
	if err != nil {
		return err
	}

	global.Logger.Info("Obtaining channel config...")
	sysch, err := GetSystemChannel(net.ID)
	if err != nil {
		return err
	}
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
		org.MSPID)
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}


	global.Logger.Info("sign for org_update_in_envelope.pb")
	UpdateSDK(n.ID)
	sdk, err := GetSDKByNetWorkID(n.ID)
	if err != nil {
		return err
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(fmt.Sprintf("ca.net%d.com", n.ID)), mspclient.WithOrg("ordererorg"))
	if err != nil {
		return err
	}

	signs := []msp.SigningIdentity{}
	adminIdentity, err := mspClient.GetSigningIdentity(fmt.Sprintf("Admin1@net%d.com", n.ID))
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

	resmgmtClient, err := resmgmt.New(
		sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@net%d.com", net.ID)), fabsdk.WithOrg("ordererorg")))
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
		resmgmt.WithOrdererEndpoint(fmt.Sprintf("orderer1.net%d.com", net.ID)))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}

	return nil
}

// AddOrg creates an organizational entity
func (n *Network) AddOrg() error {
	net, err := GetNetworkfromNets(n.ID)
	if err != nil {
		return err
	}
	org := GetBasicOrg(len(net.Organizations), net.ID)
	if err := org.CreateBasicOrganizationEntity(); err != nil {
		return err
	}
	if err := org.CreateNodeEntity(); err != nil {
		return err
	}
	if err := net.AddOrgToConsortium(org.ID); err != nil {
		return err
	}

	o, _ := GetOrgFromNets(org.ID, net.ID)
	o.Status = "success"
	UpdateNets(o)

	return nil
}

func (n *Network) AddChannel(orgIDs []int) error {
	channel := Channel{
		ID: len(n.Channels) + 1,
		Name: fmt.Sprintf("channel%d", len(n.Channels) + 1),
		NetworkID: n.ID,
		Organizations: []Organization{},
		Orderers: []Order{n.Orders[0]},
		Status: "starting",
	}

	orgNum := len(n.Organizations) - 1
	for _, id := range orgIDs {
		if id < 1 || id > orgNum {
			return errors.New(fmt.Sprintf("org%d does not exist in the network", id))
		}
	}
	channel.Organizations = append(channel.Organizations, n.Organizations[orgIDs[0]])


	if err := channel.RenderConfigtx(); err != nil {
		return errors.WithMessage(err, "fail to render configtx.yaml")
	}

	tools := kubernetes.Tools{}
	global.Logger.Info("generate a default channel")
	_, _, err := tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "NewChannel",
		"-channelID", fmt.Sprintf("channel%d", channel.ID),
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel%d.tx", n.ID, channel.ID),
	)
	if err != nil {
		return err
	}

	if err := channel.CreateChannel(n.Orders[0].Name); err != nil {
		return err
	}

	for _, peer := range channel.Organizations[0].Peers {
		peer.JoinChannel(fmt.Sprintf("channel%d", channel.ID), channel.Orderers[0].Name)
	}

	for i := 1; i < len(orgIDs); i++ {
		global.Logger.Info(fmt.Sprintf("add org%d to channel%d", orgIDs[i], channel.ID))
		if err := channel.AddOrg(orgIDs[i]); err != nil {
			return err
		}
	}

	c, err := GetChannelFromNets(channel.ID, channel.NetworkID)
	if err != nil {
		global.Logger.Error("fail to get channel ", zap.Error(err))
	}
	for i := 1; i < len(channel.Organizations); i++ {
		org := channel.Organizations[i]
		for _, peer := range org.Peers {
			if err := peer.JoinChannel(fmt.Sprintf("channel%d", c.ID), c.Orderers[0].Name); err != nil {
				global.Logger.Error(fmt.Sprintf("%s fail to join channel%d", peer.Name, c.ID))
			}
		}
	}

	return nil
}

func (n *Network)RefreshChannels() error {
	global.Logger.Info(fmt.Sprintf("Refresh channels in net%d", n.ID))
	for _, ch := range n.Channels {
		global.Logger.Info(fmt.Sprintf("update channel%d...", ch.ID))
		newCh, err := GetChannelFromNets(ch.ID, ch.NetworkID)
		if err != nil {
			return errors.WithMessage(err, "fail to refresh channels")
		}
		UpdateNets(*newCh)
	}
	return nil
}


// 给network中的自定义字段使用
// scan for scanner helper
func scan(data interface{}, value interface{}) error {
	if value == nil {
		return nil
	}

	switch value.(type) {
	case []byte:
		return json.Unmarshal(value.([]byte), data)
	case string:
		return json.Unmarshal([]byte(value.(string)), data)
	default:
		return fmt.Errorf("val type is valid, is %+v", value)
	}
}

// for valuer helper
func value(data interface{}) (interface{}, error) {
	vi := reflect.ValueOf(data)
	// 判断是否为 0 值
	if vi.IsZero() {
		return nil, nil
	}
	return json.Marshal(data)
}
