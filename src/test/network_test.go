package test

import (
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"mictract/global"
	"mictract/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testNet = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{
		{
			Name: "orderer1.net1.com",
		},
	},
	Organizations: []model.Organization{
		{
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
/*
func TestCreateNetwork(t *testing.T) {
	tests := []struct {
		Net  model.Network
		Code int
	}{
		{Net: testNet, Code: 200},
		{Net: model.Network{}, Code: 400},
	}

	for _, tc := range tests {
		w := Post("/network/", Parse(tc.Net))

		fmt.Println(w.Body.String())
		assert.Equal(t, tc.Code, w.Code)
	}
}

func TestListNetworks(t *testing.T) {
	w := Get("/network/", Parse(request.PageInfo{
		Page:     1,
		PageSize: 2,
	}))

	fmt.Println(w.Body.String())
	assert.Equal(t, 200, w.Code)
}
*/
func TestDeployNetwork(t *testing.T) {
	var err error
	if err = testNet.Deploy(); err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}

	assert.Equal(t, true, err == nil)
}

func TestRenderConfigtx(t *testing.T) {
	var err error
	if err = testNet.RenderConfigtx(); err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}

	assert.Equal(t, true, err == nil)
}
func TestGetSDK(t *testing.T) {
	// testNet.GetSDK()

	sdkconfig, err := yaml.Marshal(model.NewSDKConfig(&testNet))
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string((sdkconfig)))

	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		fmt.Println(err.Error())
	}
	global.SDKs[testNet.Name] = sdk
}

func TestGetCAInfo(t *testing.T) {
	sdk, err := testNet.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance("ca.org1.net1.com"), mspclient.WithOrg("org1"))
	if err != nil {
		global.Logger.Error("fail to get mspCient", zap.Error(err))
	}

	cainfo, err := mspClient.GetCAInfo()
	if err != nil {
		global.Logger.Error("fail to get CAInfo", zap.Error(err))
	}
	fmt.Println(string(cainfo.CAChain))
}