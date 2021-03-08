package test

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"mictract/config"
	"mictract/global"
	"mictract/model"
	"os"
	"path/filepath"
	"testing"
)

//
var n = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{
		{"orderer1.net1.com"},
	},
	Channels: []model.Channel{
		model.Channel{
			ID: 1,
			Name: "channel1",
			NetworkID: 1,
			Organizations: []model.Organization{
				{
					ID: 1,
					Name: "org1",
					NetworkID: 1,
					MSPID: "org1MSP",
					Peers: []model.Peer{
						{"peer1.org1.net1.com"},
					},
					Users: []string{
						"Admin1@org1.net1.com",
					},
				},
			},
			Orderers: []model.Order{
				{"orderer1.net1.com"},
			},
		},
	},
	Organizations: []model.Organization{
		{
			ID: -1,
			NetworkID: 1,
			Name: "ordererorg",
			Peers: []model.Peer{
				{"orderer1.net1.com"},
			},
			Users: []string{
				"Admin1@net1.com",
				"User1@net1.com",
			},
		},
		{
			ID: 1,
			NetworkID: 1,
			Name: "org1",
			Peers: []model.Peer{
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


func TestJoinChannel(t *testing.T) {
	//adminUser := "Admin1@org1.net1.com"
	//orgName := "org1"
	ordererURL := "orderer1.net1.com"

	sdk, err := n.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	//channelConfigTxPath := filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", n.ID), channelName + ".tx")



	rcp := sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin1@org1.net%d.com", n.ID)), fabsdk.WithOrg("org1"))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		global.Logger.Error("fail to get rc", zap.Error(err))
		//return errors.WithMessage(err, "fail to get rc ")
	}

	if err := rc.JoinChannel(
		"channel1",
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(ordererURL)); err != nil {
		global.Logger.Error("fail to joinchannel", zap.Error(err))
	}

}

func TestWithUser(t *testing.T) {
	sdk, err := n.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	defer sdk.Close()

	id, err := sdk.Context(fabsdk.WithUser("Admin1@org1.net1.com"), fabsdk.WithOrg("org1"))()
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(string(id.EnrollmentCertificate()))
}

func TestGetChannelConfig(t *testing.T)  {
	model.UpdateNets(n)
	net, _:= model.GetNetworkfromNets(1)
	channel := net.Channels[0]

	bt, err := channel.GetChannelConfig()
	if err != nil {
		fmt.Println(err.Error())
	}
	f, _ := os.Create(filepath.Join(config.LOCAL_BASE_PATH, "config_block.pb"))
	f.Write(bt)
	f.Close()
}

func TestChannelAddOrg(t *testing.T) {
	model.UpdateNets(n)
	net, _:= model.GetNetworkfromNets(1)

	channel := net.Channels[0]
	if err := channel.AddOrg(2); err != nil {
		fmt.Println(err.Error())
	}
}

func TestChannelUpdateAnchors(t *testing.T) {
	model.UpdateNets(n)
	net, _ := model.GetNetworkfromNets(1)

	channel := net.Channels[0]
	if err := channel.UpdateAnchors(1); err != nil {
		fmt.Println(err.Error())
	}
}

