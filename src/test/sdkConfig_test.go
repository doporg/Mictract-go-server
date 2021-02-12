package test

import (
	"fmt"
	"mictract/model"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSDKConfig(t *testing.T) {
	// 5 orderer 2 org 4 peer 3ca
	net := model.Network{
		Name: "net1",
		Orders: []model.Order{
			{
				Name: "orderer1.net1.com",
			},
			{
				Name: "orderer2.net1.com",
			},
			{
				Name: "orderer3.net1.com",
			},
			{
				Name: "orderer4.net1.com",
			},
			{
				Name: "orderer5.net1.com",
			},
		},
		Organizations: []model.Organization{
			{
				Name: "org1",
				Peers: []model.Peer{
					{
						Name: "peer1.org1.net1.com",
					},
					{
						Name: "peer2.org1.net1.com",
					},
				},
			},
			{
				Name: "org2",
				Peers: []model.Peer{
					{
						Name: "peer1.org2.net1.com",
					},
					{
						Name: "peer2.org2.net1.com",
					},
				},
			},
		},
		Consensus:  "solo",
		TlsEnabled: true,
	}

	sdkconfig := model.NewSDKConfig(&net)
	out, err := yaml.Marshal(sdkconfig)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(out))
}
