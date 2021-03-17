package test

import (
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
	"testing"
)
var nn = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{},
	Channels: []model.Channel{},
	Organizations: []model.Organization{
		{
			ID: -1,
			NetworkID: 1,
			Name: "ordererorg",
			MSPID: "ordererMSP",
			Peers: []model.Peer{
				{"orderer1.net1.com"},
			},
			Users: []string{
				"Admin1@net1.com",
			},
		},
		{
			ID: 1,
			NetworkID: 1,
			MSPID: "org1MSP",
			Name: "org1",
			Peers: []model.Peer{
				{"peer1.org1.net1.com"},
			},
			Users: []string{
				"Admin1@org1.net1.com",
			},
		},
	},
	Consensus:  "solo",
	TlsEnabled: true,
}
var org2 = model.Organization{
	ID: 2,
	Name: "org2",
	NetworkID: 1,
	MSPID: "org2MSP",
	Peers: []model.Peer{},
	Users: []string{},
}
func TestCreateEntity(t *testing.T) {
	model.UpdateNets(nn)
	if err := org2.CreateBasicOrganizationEntity(); err != nil {
		global.Logger.Error("11111111111111111111111", zap.Error(err))
	}
	if err := org2.CreateNodeEntity(); err != nil {
		global.Logger.Error("22222222222222222222222", zap.Error(err))
	}
}

func TestRemoveAllEntity(t *testing.T)  {
	org2 = model.Organization{
		ID: 2,
		Name: "org2",
		NetworkID: 1,
		MSPID: "org2MSP",
		Peers: []model.Peer{
			{"peer1.org2.net1.com"},
		},
		Users: []string{},
	}
	org2.RemoveAllEntity()
}