package test

import (
	"fmt"
	"mictract/global"
	ii "mictract/init"
	"mictract/model"
	"testing"
)

func TestNetworkCRUD(t *testing.T) {
	global.Logger.Info("start test ....")
	for i := 0; i < 10; i++ {
		model.UpdateNets(*model.GetBasicNetwork("solo"))
	}
	model.DeleteNetworkByID(1)
	model.UpdateNets(*model.GetBasicNetwork("solo"))
	global.Logger.Info("db will close...")
	ii.Close()
}

func TestStore(t *testing.T) {
	var testNet = model.Network{
		ID: 1,
		Name: "net1",
		Orders: model.Orders{
			{"orderer1.net1.com"},
			{"orderer2.net1.com"},
		},
		Channels: model.Channels{
			{
				ID: 1,
				Name: "channel1",
				NetworkID: 1,
				Organizations: model.Organizations{
					{
						ID: -1,
						NetworkID: 1,
						Name: "ordererorg",
						MSPID: "ordererMSP",
						Peers: []model.Peer{
							{"peer1.org1.net1.com"},
						},
						Users: []string{
							"Admin1@org1.net1.com",
							"User1@org1.net1.com",
						},
					},
				},
				Orderers: model.Orders{
					{"Orderer1.net1.com"},
				},
			},
		},
		Organizations: model.Organizations{
			{
				ID: -1,
				NetworkID: 1,
				Name: "ordererorg",
				MSPID: "ordererMSP",
				Peers: model.Peers{
					{"peer1.org1.net1.com"},
				},
				Users: []string{
					"Admin1@org1.net1.com",
					"User1@org1.net1.com",
				},
			},
		},
		Consensus:  "solo",
		TlsEnabled: true,
	}
	if err := testNet.Insert(); err != nil {
		fmt.Println(err.Error())
	}
}

func TestScan(t *testing.T) {
	nets, err := model.QueryAllNetwork()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(nets)
	}
}