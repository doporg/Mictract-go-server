package test

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"log"
	"mictract/global"
	"mictract/model"
	"testing"
	_ "mictract/init"
)

func TestNewChaincode(t *testing.T) {
	var err error
	_, err = model.NewChaincode([]byte("new chaincode"), "genisisCC", "golang")
	if err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}
	assert.Equal(t, true, err == nil)
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
