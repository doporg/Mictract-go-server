package model

import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
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
	"text/template"
	"time"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"

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

	SDK *fabsdk.FabricSDK
}

var (
	// just demo
	// one orderer one org one peer
	networks = []Network{
		{
			Name: "net1",
			Orders: []Order{
				{
					Name: "orderer.net1.com",
				},
			},
			Organizations: []Organization{
				{
					Name: "org1",
					Peers: []Peer{
						{
							Name: "peer1.org1.net1.com",
						},
					},
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

// Deploy method is just creating a basic network containing only 1 peer and 1 orderer,
//	and then join the rest of peers and orderers.
// The basic network is built to make `configtx.yaml` file simple enough to create the genesis block.
func (n *Network) Deploy() (err error) {
	// create ca and tools resources
	tools := kubernetes.Tools{}
	models := []kubernetes.K8sModel{
		&tools,
		kubernetes.NewPeerCA(n.ID, 1),
		kubernetes.NewOrdererCA(n.ID),
	}

	for _, m := range models {
		m.Create()
	}

	// TODO: make it sync
	// wait for pulling images when first deploy
	time.Sleep(30 * time.Second)

	// call CaUser.GenerateOrgMsp for GetSDK
	causers := []CaUser {
		{
			OrganizationID: -1,
			NetworkID: n.ID,
		},
		{
			OrganizationID: 1,
			NetworkID: n.ID,
		},
	}
	for _, causer := range causers {
		if err := causer.GenerateOrgMsp(); err != nil {
			return err
		}
	}

	var sdk *fabsdk.FabricSDK
	if sdk, err = n.GetSDK(); err != nil {
		return err
	}

	// create an organization
	{
		var mspClient *mspclient.Client
		caUrl := fmt.Sprintf("ca.org1.net%d.com", n.ID)
		if mspClient, err = mspclient.New(sdk.Context(), mspclient.WithCAInstance(caUrl), mspclient.WithOrg("Org1")); err != nil {
			return err
		}

		// register users of this organization
		users := []*CaUser{
			NewUserCaUser(1, 1, n.ID, "user1pw"),
			NewAdminCaUser(1, 1, n.ID, "admin1pw"),
			NewPeerCaUser(1, 1, n.ID, "peer1pw"),
		}

		for _, u := range users {
			if err = u.Register(mspClient); err != nil {
				return err
			}
		}

		// enroll to build msp and tls directories
		for _, u := range users {
			// msp
			if err = u.Enroll(mspClient, false); err != nil {
				return err
			}
			// tls
			if err = u.Enroll(mspClient, n.TlsEnabled); err != nil {
				return err
			}
		}
	}

	// create an orderer organization
	{
		var mspClient *mspclient.Client
		caUrl := fmt.Sprintf("ca.net%d.com", n.ID)
		if mspClient, err = mspclient.New(sdk.Context(), mspclient.WithCAInstance(caUrl), mspclient.WithOrg("OrdererOrg")); err != nil {
			return err
		}

		// register users of this organization
		users := []*CaUser{
			NewUserCaUser(1, -1, n.ID, "user1pw"),
			NewAdminCaUser(1, -1, n.ID, "admin1pw"),
			NewOrdererCaUser(1, n.ID, "orderer1pw"),
		}

		for _, u := range users {
			if err = u.Register(mspClient); err != nil {
				return err
			}
		}

		// enroll to build msp and tls directories
		for _, u := range users {
			// msp
			if err = u.Enroll(mspClient, false); err != nil {
				return err
			}
			// tls
			if err = u.Enroll(mspClient, n.TlsEnabled); err != nil {
				return err
			}
		}
	}

	// configtx.yaml should be placed in `networks/netX/configtx.yaml`
	if err = n.RenderConfigtx(); err != nil {
		return err
	}

	// generate the genesis block
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "Genesis",
		"-channelID", "system-channel",
		"-outputBlock", fmt.Sprintf("/mictract/networks/net%d/genesis.block", n.ID),
	)

	if err != nil {
		return err
	}

	// generate a default channel
	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", n.ID),
		"-profile", "DefaultChannel",
		"-channelID", "channel1",
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel1.tx", n.ID),
	)

	if err != nil {
		return err
	}

	// create rest of resources
	models = []kubernetes.K8sModel{
		kubernetes.NewOrderer(n.ID, 1),
		kubernetes.NewPeer(n.ID, 1, 1),
	}

	for _, m := range models {
		m.Create()
	}

	// Create first Channel channl1
	if err := n.UpdateSDK(); err != nil {
		return err
	}
	_, err = GetSDKByNetWorkID(1)
	if err != nil {
		return err
	}
	if err := n.CreateChannel("channel1", "orderer1.net1.com"); err != nil {
		return errors.WithMessage(err, "fail to create channel")
	}

	// TODO: join peers into the first channel
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

func (n *Network) UpdateSDK() error {
	sdkconfig, err := yaml.Marshal(NewSDKConfig(n))
	if err != nil {
		return err
	}
	global.Logger.Info(string(sdkconfig))
	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		return err
	}
	global.SDKs[n.Name] = sdk
	return nil
}

func (n *Network) GetSDK() (*fabsdk.FabricSDK, error) {
	if _, ok := global.SDKs[n.Name]; !ok {
		if err := n.UpdateSDK(); err != nil {
			return nil, err
		}
	}
	return global.SDKs[n.Name], nil
}

func GetSDKByNetWorkID(id int) (*fabsdk.FabricSDK, error) {
	global.Logger.Info("current SDK:")
	for k, _ := range global.SDKs {
		global.Logger.Info(k)
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



func (n *Network)GetAllAdminSigningIdentities() ([]msp.SigningIdentity, error) {
	signs := []msp.SigningIdentity{}
	// peer admin (n.Orgnaizations include ordererorg
	for _, org := range n.Organizations {
		username := fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, n.ID)
		if org.ID == -1 {
			username = fmt.Sprintf("Admin1@net%d.com", n.ID)
		}
		password := "admin1pw"

		mspClient, err := org.NewMspClient()
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d mspClient", org.ID), zap.Error(err))
		}

		if err := mspClient.Enroll(username, mspclient.WithSecret(password)); err != nil {
			global.Logger.Error("fail to enroll user " + username, zap.Error(err))
		}

		sign, err := mspClient.GetSigningIdentity(username)
		if err != nil {
			global.Logger.Error(fmt.Sprintf("fail to get org%d AdminSigningIdentity", org.ID), zap.Error(err))
		}
		signs = append(signs, sign)
	}

	return signs, nil
}


func (n *Network)CreateChannel(channelName, ordererURL string) error {
	sdk, err := n.GetSDK()
	if err != nil {
		return errors.WithMessage(err, "fail to get sdk ")
	}
	channelConfigTxPath := filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", n.ID), channelName + ".tx")

	adminIdentitys, err := n.GetAllAdminSigningIdentities()
	if err != nil {
		return errors.WithMessage(err, "fail to get all SigningIdentities")
	}
	req := resmgmt.SaveChannelRequest{
		ChannelID: channelName,
		ChannelConfigPath: channelConfigTxPath,
		SigningIdentities: adminIdentitys,
	}

	rcp := sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@org1.net%d.com", n.ID)), fabsdk.WithOrg("org1"))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}
	_, err = rc.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint(ordererURL))
	return err
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
