package model

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	"gorm.io/gorm"
)

type Network struct {
	gorm.Model
	ID            int           `json:"id"`
	Name          string        `json:"name" binding:"required"`
	Orders        Orders        `json:"orders" binding:"required"`
	Organizations Organizations `json:"organizations" binding:"required"`
	Channels      Channels      `json:"channels"`

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
	if end > len(networks) {
		end = len(networks)
	}

	return networks[start:end], nil
}

func FindNetworkByID(id int) (Network, error) {
	// TODO
	for _, n := range networks {
		if id == n.ID {
			return n, nil
		}
	}

	return Network{}, fmt.Errorf("network not found")
}

func DeleteNetworkByID(id int) error {
	// TODO
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
func GetBasicNetwork() *Network {
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
		Name: fmt.Sprintf("net%d", len(global.Nets) + 1),
		Orders: []Order{},
		Organizations: []Organization{},
		Channels: []Channel{},
		Consensus: "solo",
		TlsEnabled: true,
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

	ordererOrg := Organization{
		ID: -1,
		NetworkID: n.ID,
		Name: "ordererorg",
		MSPID: "ordererMSP",
		Peers: []Peer{},
		Users: []string{},
	}
	org1 := Organization{
		ID: 1,
		NetworkID: n.ID,
		Name: "org1",
		MSPID: "org1MSP",
		Peers: []Peer{},
		Users: []string{},
	}
	channel := Channel{
		ID: 1,
		Name: "channel1",
		NetworkID: n.ID,
		Organizations: Organizations{
			ordererOrg,
			org1,
		},
	}

	// create tools resources
	global.Logger.Info("Start the tools node")
	tools := kubernetes.Tools{}
	tools.Create()


	// TODO: make it sync
	// wait for pulling images when first deploy
	time.Sleep(5 * time.Second)

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
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "DefaultChannel",
		"-channelID", "channel1",
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel1.tx", n.ID),
	)

	global.Logger.Info("此处需要同步，如果你看到这条信息，不要忘了增加同步代码，并且删除这条info")
	// TODO: make it sync
	// wait for pulling images when first deploy
	time.Sleep(30 * time.Second)


	// Create first Channel channl1
	if err := channel.CreateChannel(fmt.Sprintf("orderer1.net%d.com", n.ID)); err != nil {
		return errors.WithMessage(err, "fail to create channel")
	}

	// TODO: join peers into the first channel
	n, err = GetNetworkfromNets(n.ID)
	if err != nil {
		global.Logger.Error("Unable to get the latest network", zap.Error(err))
	}
	if err := n.Organizations[1].Peers[0].JoinChannel("channel1", fmt.Sprintf("orderer1.net%d.com", n.ID)); err != nil {
		return errors.WithMessage(err, "fail to join channel")
	}

	// TODO: create the rest of organizations

	return nil
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
			n.Organizations[cu.OrganizationID].Peers = append(n.Organizations[cu.OrganizationID].Peers, Peer{Name: cu.GetUsername()})
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
func (n *Network)AddOrderersToSystemChannel() error {
	global.Logger.Info("Add Orderer to system-channel ...")
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
	newOrderer := Order{
		Name: fmt.Sprintf("orderer%d.net%d.com", newOrdererID, n.ID),
	}
	n.Orders = append(n.Orders, newOrderer)


	UpdateSDK(n.ID)
	sdk, err := GetSDKByNetWorkID(n.ID)
	if err != nil {
		return err
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(fmt.Sprintf("ca-net%d", n.ID)), mspclient.WithOrg("ordererorg"))
	if err != nil {
		return err
	}
	// regiester new orderer
	user := NewOrdererCaUser(newOrdererID, n.ID, fmt.Sprintf("orderer%dpw", newOrdererID))
	if err := user.Register(mspClient); err != nil {
		return err
	}

	// enroll new orderer
	if err := user.Enroll(mspClient, true); err != nil {
		return err
	}
	if err := user.Enroll(mspClient, false); err != nil {
		return err
	}

	// generate ord1.json
	st := `["`
	for _, orderer := range n.Orders {
		user := NewCaUserFromDomainName(orderer.Name)
		tlscert := user.GetTLSCert(true)
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) +
			`","host":"` + user.GetURL() +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(tlscert)) + `"},`
	}
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
		fmt.Sprintf("channel%d", c.ID),
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
	return c.updateConfig(signs)
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
