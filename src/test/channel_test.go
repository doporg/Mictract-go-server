package test

import (
	"fmt"
	"mictract/model"
	"testing"
)

var n = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{
		{
			Name: "orderer1.net1.com",
		},
	},
	Organizations: []model.Organization{
		{
			ID: 1,
			NetworkID: 1,
			Name: "org1",
			Peers: []model.Peer{
				{
					Name: "peer1.org1.net1.com",
				},
			},
		},
	},
	Consensus:  "solo",
	TlsEnabled: true,
}

func TestCreateChannel(t *testing.T) {
	if err := n.UpdateSDK(); err != nil {
		fmt.Println(err.Error())
	}
	_, err := model.GetSDKByNetWorkID(1)
	if err != nil {
		fmt.Println(err.Error())
	}
	if err := n.CreateChannel("channel1", "orderer1.net1.com"); err != nil {
		fmt.Println(err.Error())
	}
}