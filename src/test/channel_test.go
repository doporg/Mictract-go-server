package test

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
	"testing"
)

var n = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{},
	Channels: []model.Channel{},
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
			},
		},
	},
	Consensus:  "solo",
	TlsEnabled: true,
}

func TestCreateChannel(t *testing.T) {
	if err := n.CreateChannel("orderer1.net1.com"); err != nil {
		fmt.Println(err.Error())
	}
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

func TestChannelAddOrg(t *testing.T) {
	model.UpdateNets(n)
	channel := model.Channel{
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
	}
	org := model.Organization{
		ID: 2,
		NetworkID: 1,
		Name: "org2",
		MSPID: "org2MSP",
		Peers: []model.Peer{
			{"peer1.org2.net1.com"},
		},
		Users: []string{
			"Admin1@org2.net1.com",
		},
	}
	if err := channel.AddOrg(&org); err != nil {
		fmt.Println(err.Error())
	}
	//ledgerClient, err := channel.NewLedgerClient(channel.Organizations[0].Users[0], channel.Organizations[0].Name)
	//if err != nil {
	//	global.Logger.Error("fail to get ledgerClient", zap.Error(err))
	//}
	//blk, err := ledgerClient.QueryConfigBlock(ledger.WithTargetEndpoints(channel.Organizations[0].Peers[0].Name))
	//if err != nil {
	//	global.Logger.Error("fail go get configBlock", zap.Error(err))
	//}
	//fmt.Println(blk.String())
}

