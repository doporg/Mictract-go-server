package model

import (
	"encoding/json"
	"fmt"
	"mictract/global"
	"mictract/model/request"
	"reflect"

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
	Consensus     string        `json:"consensus" binding:"required"`
	TlsEnabled    bool          `json:"tlsEnabled"`
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

func (n *Network) Deploy() {
	// TODO
	// generate fabric-ca configurations and send them to k8s
	// enroll admin and register users to generate MSPs
	// generate order system and genesis block
	// generate organizations configurations and send them to k8s
	// create channel
	// join all peers into the channel
	// set the anchor peers for each org
}

func (n *Network) GetSDK() (*fabsdk.FabricSDK, error) {
	if _, ok := global.SDKs[n.Name]; !ok {
		sdkconfig, err := yaml.Marshal(NewSDKConfig(n))
		if err != nil {
			return nil, err
		}
		sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
		if err != nil {
			return nil, err
		}
		global.SDKs[n.Name] = sdk
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
