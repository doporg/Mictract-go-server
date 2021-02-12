package model

import (
	"fmt"
	"mictract/global"
	"mictract/model/request"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"gopkg.in/yaml.v3"
)

type Network struct {
	ID            int            `json:"id"`
	Name          string         `json:"name" binding:"required"`
	Orders        []Order        `json:"orders" binding:"required"`
	Organizations []Organization `json:"organizations" binding:"required"`
	Consensus     string         `json:"consensus" binding:"required"`
	TlsEnabled    bool           `json:"tlsEnabled"`
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

func (n *Network) GenerateSDKConfig() ([]byte, error) {
	return nil, nil
}
