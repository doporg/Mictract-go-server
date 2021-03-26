package test

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"log"
	"mictract/global"
	_ "mictract/init"
	"mictract/model"
	"mictract/model/kubernetes"
	"testing"
)

var (
	netID 			= 1
	channelID 		= 1
	label			= "mycc"
	excc			= true
	address			= ""
	version			= "1"
	sequence		= 1
	initRequired	= true
	policyStr		= "OR('org1MSP.member')"
	ccID			= 1

	org1Admin		= "Admin1@org1.net1.com"
	org1Name		= "org1"
	peer1Org1		= model.Peer{"peer1.org1.net1.com"}

	packageID       = "mycc:b26ec5164dca00544ecf6115e3deec9cbbc6179bd47223bbbacdcba626f86d21"
)

// 内部使用
func TestNewChaincode(t *testing.T) {
	var err error
	_, err = model.NewChaincode([]byte(""), "genisisCC", "golang")
	if err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
}

func TestPackege(t *testing.T) {

}

func TestQueryInstalled(t *testing.T) {
	admin := "Admin1@org1.net1.com"
	orgName := "org1"

	model.UpdateSDK(1)
	sdk, err := model.GetSDKByNetWorkID(1)

	rcp := sdk.Context(fabsdk.WithUser(admin), fabsdk.WithOrg(orgName))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		log.Panicf("failed getting admin user session for org: %s", err)
	}

	peer := model.Peer{
		Name: "peer1.org1.net1.com",
	}

	if resps, err := peer.QueryInstalled(rc); err != nil {
		global.Logger.Error("testing ", zap.Error(err))
	} else {
		fmt.Println(resps)
	}

	assert.Equal(t, true, err == nil)
}

func TestDeleteChaincode(t *testing.T)  {
	ccIDs := []int{1, 2}
	for _, ccID := range ccIDs {
		var err error
		if err = model.DeleteChaincodeByID(ccID); err != nil {
			global.Logger.Error("testing ", zap.Error(err))
		}
		assert.Equal(t, true, err == nil)
	}
}

// 内部使用
func TestInstallCC(t *testing.T) {
	global.Logger.Info("Obtaining cc...")
	cc, err := model.GetChaincodeByID(1)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining cci...")
	cci, err := cc.NewChaincodeInstance(
		netID, channelID, label, address, policyStr, version,
		int64(sequence),
		excc, initRequired)
	if err != nil {
		global.Logger.Error("fail to get cci", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
	global.Logger.Info(fmt.Sprintf("%v", cci))

	global.Logger.Info("Obtaining sdk...")
	model.UpdateSDK(netID)
	sdk, err := model.GetSDKByNetWorkID(netID)
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining org1 rc...")
	org1rc, err := resmgmt.New(
		sdk.Context(
			fabsdk.WithUser(org1Admin),
			fabsdk.WithOrg(org1Name)))
	if err != nil {
		global.Logger.Error("fail to get org1's rc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Installing cc...")
	if err := cci.InstallCC(org1rc); err != nil {
		global.Logger.Error("fail to install cc to org1", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("query install ...")
	resps, err := peer1Org1.QueryInstalled(org1rc)
	if err != nil {
		global.Logger.Error("testing ", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info(fmt.Sprintf("%v", resps))
}

func TestBuildCC(t *testing.T)  {
	global.Logger.Info("Obtaining cc...")
	cc, err := model.GetChaincodeByID(1)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining cci...")
	cci, err := cc.NewChaincodeInstance(
		netID, channelID, label, address, policyStr, version,
		int64(sequence),
		excc, initRequired)
	if err != nil {
		global.Logger.Error("fail to get cci", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
	global.Logger.Info(fmt.Sprintf("%v", cci))

	err = cci.Build()
	if err != nil {
		global.Logger.Error("fail to build cc ", zap.Error(err))
	}

	assert.Equal(t, true, err == nil)
}

func TestCommitCC(t *testing.T)  {


	global.Logger.Info("Obtaining cc...")
	cc, err := model.GetChaincodeByID(1)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining cci...")
	cci, err := cc.NewChaincodeInstance(
		netID, channelID, label, address, policyStr, version,
		int64(sequence),
		excc, initRequired)
	if err != nil {
		global.Logger.Error("fail to get cci", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
	global.Logger.Info(fmt.Sprintf("%v", cci))

	global.Logger.Info("Obtaining sdk...")
	model.UpdateSDK(netID)
	sdk, err := model.GetSDKByNetWorkID(netID)
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining org1 rc...")
	org1rc, err := resmgmt.New(
		sdk.Context(
			fabsdk.WithUser(org1Admin),
			fabsdk.WithOrg(org1Name)))
	if err != nil {
		global.Logger.Error("fail to get org1's rc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)


	if m, err := cci.CheckCCCommitReadiness(org1rc, peer1Org1.Name); err != nil {
		global.Logger.Error("testing ", zap.Error(err))
	} else {
		global.Logger.Info(fmt.Sprintf("%v", m))
	}

	if err := cci.CommitCC(org1rc, "orderer1.net1.com"); err != nil {
		global.Logger.Error("fail to commit cc ", zap.Error(err))
	}
}

func TestCreateCCContainer(t *testing.T)  {
	ccc := kubernetes.NewChaincode(netID, channelID, packageID, ccID)
	ccc.Create()
}

func TestDeleteCCContainer(t *testing.T) {
	ccc := kubernetes.NewChaincode(netID, channelID, packageID, ccID)
	ccc.Delete()
}

func TestExecCC(t *testing.T) {
	global.Logger.Info("Obtaining cc...")
	cc, err := model.GetChaincodeByID(1)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining cci...")
	cci, err := cc.NewChaincodeInstance(
		netID, channelID, label, address, policyStr, version,
		int64(sequence),
		excc, initRequired)
	if err != nil {
		global.Logger.Error("fail to get cci", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
	global.Logger.Info(fmt.Sprintf("%v", cci))

	global.Logger.Info("Obtaining sdk...")
	model.UpdateSDK(netID)
	sdk, err := model.GetSDKByNetWorkID(netID)
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining channel client...")
	ccp := sdk.ChannelContext(fmt.Sprintf("channel%d", channelID), fabsdk.WithUser(org1Admin), fabsdk.WithOrg(org1Name))
	chClient, err := channel.New(ccp)
	if err != nil {
		log.Panicf("failed to create channel client: %s", err)
	}
	if err != nil {
		global.Logger.Error("fail to get channel client", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	resp, err := cci.InitCC(chClient, []string{"InitLedger"}, peer1Org1.Name)
	if err != nil {
		global.Logger.Error("fail to init ledger", zap.Error(err))
	} else {
		global.Logger.Info(fmt.Sprintf("%v", resp))
	}
	assert.Equal(t, true, err == nil)

	respb, err := cci.ExecuteCC(
		chClient,
		[]string{
			"AddLover",
			"zhangsan", "famale", "123",
			"lisi", "male", "321",
		},
		peer1Org1.Name)
	if err != nil {
		global.Logger.Error("fail to exec cc", zap.Error(err))
	} else {
		global.Logger.Info(string(respb))
	}
	assert.Equal(t, true, err == nil)

	if resp, err := cci.QueryCC(chClient, []string{"QueryAllLovers"}, peer1Org1.Name); err != nil {
		global.Logger.Error("fail to query cc", zap.Error(err))
	} else {
		global.Logger.Info(string(resp))
	}
	assert.Equal(t, true, err == nil)
}

func TestCC(t *testing.T) {
	global.Logger.Info("Obtaining cc...")
	cc, err := model.GetChaincodeByID(1)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining cci...")
	cci, err := cc.NewChaincodeInstance(
		netID, channelID, label, address, policyStr, version,
		int64(sequence),
		excc, initRequired)
	if err != nil {
		global.Logger.Error("fail to get cci", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
	global.Logger.Info(fmt.Sprintf("%v", cci))

	global.Logger.Info("Obtaining sdk...")
	model.UpdateSDK(netID)
	sdk, err := model.GetSDKByNetWorkID(netID)
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Obtaining org1 rc...")
	org1rc, err := resmgmt.New(
		sdk.Context(
			fabsdk.WithUser(org1Admin),
			fabsdk.WithOrg(org1Name)))
	if err != nil {
		global.Logger.Error("fail to get org1's rc", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("Installing cc...")
	if err := cci.InstallCC(org1rc); err != nil {
		global.Logger.Error("fail to install cc to org1", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info("query install ...")
	resps, err := peer1Org1.QueryInstalled(org1rc)
	if err != nil {
		global.Logger.Error("testing ", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	global.Logger.Info(fmt.Sprintf("%v", resps))

	if err := cci.ApproveCC(org1rc, "orderer1.net1.com", peer1Org1.Name); err != nil {
		global.Logger.Error("fail to approve CC", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)

	if resp, err := cci.CheckCCCommitReadiness(org1rc, peer1Org1.Name); err != nil {
		global.Logger.Error("fail to check cc commit readiness", zap.Error(err))
	} else {
		global.Logger.Info(fmt.Sprintf("%v", resp))
	}
	assert.Equal(t, true, err == nil)

	if err := cci.CommitCC(org1rc, "orderer1.net1.com"); err != nil {
		global.Logger.Error("fail to commit cc ", zap.Error(err))
	}
}