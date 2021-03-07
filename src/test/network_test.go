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
	"mictract/model/kubernetes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testNet = model.Network{
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
			Peers: []model.Peer{},
		},
		{
			ID: 1,
			NetworkID: 1,
			MSPID: "org1MSP",
			Name: "org1",
			Peers: []model.Peer{},
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
	if err = model.GetBasicNetwork().Deploy(); err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}

	assert.Equal(t, true, err == nil)
}

func TestDeleteBasicNetwork(t *testing.T) {
	tools := kubernetes.Tools{}
	models := []kubernetes.K8sModel{
		&tools,
		kubernetes.NewPeerCA(testNet.ID, 1),
		kubernetes.NewOrdererCA(testNet.ID),
	}

	for _, m := range models {
		m.Delete()
	}

	models = []kubernetes.K8sModel{
		kubernetes.NewOrderer(testNet.ID, 1),
		kubernetes.NewPeer(testNet.ID, 1, 1),
	}

	for _, m := range models {
		m.Delete()
	}


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

func TestRegisterOrderer(t *testing.T) {
	username := "orderer1"
	password := "ordererpw"
	usertype := "orderer"
	orgName := "ordererorg"
	caid := "ca.org1.net1.com"

	sdk, err := testNet.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance(caid), mspclient.WithOrg(orgName))
	if err != nil {
		global.Logger.Error("fail to get mspClient", zap.Error(err))
	}

	request := &mspclient.RegistrationRequest{
		Name:   username,
		Type:   usertype,
		Secret: password,
	}

	_, err = mspClient.Register(request)
	if err != nil {
		global.Logger.Error("fail to register ", zap.Error(err))
	}
}

func TestEnrollOrderer(t *testing.T) {
	username := "orderer1"
	password := "ordererpw"
	sdk, err := testNet.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance("ca.org1.net1.com"), mspclient.WithOrg("org1"))
	if err != nil {
		global.Logger.Error("fail to get mspClient", zap.Error(err))
	}

	_ = mspClient.Enroll(username, mspclient.WithSecret(password))

	resp, err := mspClient.GetSigningIdentity(username)
	if err != nil {
		global.Logger.Error("fail to get SignID ", zap.Error(err))
	}

	cert := resp.EnrollmentCertificate()
	privkey, err := resp.PrivateKey().Bytes()
	if err != nil {
		global.Logger.Error("fail to get priv", zap.Error(err))
	}

	fmt.Println("cert:")
	fmt.Println(string(cert))
	fmt.Println("priv:")
	fmt.Println(string(privkey))
}
