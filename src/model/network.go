package model

import (
	"encoding/json"
	"fmt"
	"html/template"
	mConfig "mictract/config"
	"mictract/global"
	"mictract/model/kubernetes"
	"mictract/model/request"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"

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

// TODO: need unit test
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
	time.Sleep(5 * time.Second)

	// call CaUser.GenerateOrgMsp for GetSDK
	causers := []CaUser {
		CaUser {
			OrganizationID: -1,
			NetworkID: n.ID,
		},
		CaUser {
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
		var mspClient *msp.Client
		caUrl := fmt.Sprintf("ca.org1.net%d.com", n.ID)
		if mspClient, err = msp.New(sdk.Context(), msp.WithCAInstance(caUrl), msp.WithOrg("Org1")); err != nil {
			return err
		}
/*
		enrollOptions := []msp.EnrollmentOption{
			msp.WithSecret("adminpw"),
		}

		if n.TlsEnabled {
			enrollOptions = append(enrollOptions, msp.WithProfile("tls"))
		}

		if err = mspClient.Enroll("admin", enrollOptions...); err != nil {
			return err
		}*/

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
			if err = u.Enroll(mspClient, n.TlsEnabled); err != nil {
				return err
			}
		}
	}

	// create an orderer organization
	{
		var mspClient *msp.Client
		caUrl := fmt.Sprintf("ca.net%d.com", n.ID)
		if mspClient, err = msp.New(sdk.Context(), msp.WithCAInstance(caUrl), msp.WithOrg("OrdererOrg")); err != nil {
			return err
		}
/*
		enrollOptions := []msp.EnrollmentOption{
			msp.WithSecret("adminpw"),
		}

		if n.TlsEnabled {
			enrollOptions = append(enrollOptions, msp.WithProfile("tls"))
		}

		if err = mspClient.Enroll("admin", enrollOptions...); err != nil {
			return err
		}
*/
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
