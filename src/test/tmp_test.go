package test

import (
	"fmt"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"testing"
)
var net = model.Network{
	ID: 1,
	Name: "net1",
	Orders: []model.Order{
		{"orderer1.net1.com"},
	},
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
func TestFuck(t *testing.T) {
	fmt.Println(model.NewCaUserFromDomainName("orderer1.net1.com"))
}
func TestGenerateConfigtx(t *testing.T) {
	tools 		:= &kubernetes.Tools{}
	if err := net.RenderConfigtx(); err != nil {
		global.Logger.Error("fail to exec RenderConfigtx", zap.Error(err))
	}

	// generate the genesis block
	// configtxgen -configPath /mictract/networks/net1 -profile Genesis -channelID system-channel -outputBlock /mictract/networks/net1/genesis.block
	_, _, err := tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", net.ID),
		"-profile", "Genesis",
		"-channelID", "system-channel",
		"-outputBlock", fmt.Sprintf("/mictract/networks/net%d/genesis.block", net.ID),
	)

	if err != nil {
		global.Logger.Error("fail to generate genesis.block", zap.Error(err))
	}

	_, _, err = tools.ExecCommand("configtxgen",
		"-configPath", fmt.Sprintf("/mictract/networks/net%d/", net.ID),
		"-profile", "DefaultChannel",
		"-channelID", "channel1",
		"-outputCreateChannelTx", fmt.Sprintf("/mictract/networks/net%d/channel1.tx", net.ID),
	)

	if err != nil {
		global.Logger.Error("fail to generate channel.tx", zap.Error(err))
	}
}
func TestRegiesterUser(t *testing.T) {
	username := "User1@org1.net1.com"
	password := "userpw"
	usertype := "user"
	orgName := "org1"
	caid := "ca.org1.net1.com"

	sdk, err := net.GetSDK()
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

func TestEnrollUser(t *testing.T) {
	username := "User1@org1.net1.com"
	             //User1@org1.net1.com
	password := "user1pw"
	sdk, err := net.GetSDK()
	if err != nil {
		global.Logger.Error("fail to get sdk", zap.Error(err))
	}

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance("ca.org1.net1.com"), mspclient.WithOrg("org1"))
	if err != nil {
		global.Logger.Error("fail to get mspClient", zap.Error(err))
	}

	mspClient.Enroll(username, mspclient.WithSecret(password))

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

func TestCreateCA(t *testing.T) {
	kubernetes.NewPeerCA(net.ID, 1).Create()
}
func TestDeleteCA(t *testing.T) {
	kubernetes.NewPeerCA(net.ID, 1).Delete()
}
func TestCreateOrderer(t *testing.T) {
	kubernetes.NewOrderer(1, 1).Create()
}
func TestDeleteOrderer(t *testing.T) {
	kubernetes.NewOrderer(1, 1).Delete()
}

func TestGetSign(t *testing.T) {
	org := model.Organization{
		ID: 1,
		Name: "org1",
		MSPID: "org1MSP",
		NetworkID: 1,
	}
	/*out, _ := yaml.Marshal(model.NewSDKConfig(&net))
	fmt.Println(string(out))


	sdk, _ := fabsdk.New(config.FromRaw(out, "yaml"))

	mspClient, err := mspclient.New(sdk.Context(), mspclient.WithCAInstance("ca.org1.net1.com"), mspclient.WithOrg("org1"))*/
	model.UpdateSDK(net.ID)
	mspClient, err := org.NewMspClient()
	if err != nil {
		fmt.Println("llj:", err.Error())
	}

	username := fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, net.ID)
	fmt.Println("username: " + username)
	err = mspClient.Enroll(username, mspclient.WithSecret("admin1pw"))
	if err != nil {
		fmt.Println(err.Error() + "llj")
	}
	sign, err := mspClient.GetSigningIdentity(username)
	if err != nil {
		global.Logger.Error(fmt.Sprintf("fail to get org%d AdminSigningIdentity", org.ID), zap.Error(err))
	}

	priv, _:= sign.PrivateKey().Bytes()
	fmt.Println(string(priv))
}

func TestGetCertAndPriv(t *testing.T) {
	sdk, _ := net.GetSDK()

	mspClient, _ := mspclient.New(sdk.Context(), mspclient.WithCAInstance("ca.org1.net1.com"), mspclient.WithOrg("org1"))

	//user := model.NewCaUserFromDomainName("peer1.org1.net1.com")
	err := mspClient.Enroll("peer1.org1.net1.com", mspclient.WithSecret("peer1pw") ,mspclient.WithCSR(&mspclient.CSRInfo{
		Hosts: []string{"peer1-org1-net1"},
	}))
	if err != nil {
		fmt.Println(err.Error())
	}

	resp, err := mspClient.GetSigningIdentity("peer1.org1.net1.com")
	if err != nil {
		fmt.Println("fuck msp", err.Error())
	}

	priv, err:= resp.PrivateKey().Bytes()
	publ := resp.EnrollmentCertificate()

	fmt.Println(string(priv))
	fmt.Println(string(publ))
}

func TestGetOrdererAdminSign(t *testing.T) {
	model.UpdateSDK(net.ID)
	// orderer admin
	username := fmt.Sprintf("Admin1@net%d.com", 1)
	password := "admin1pw"
	org := model.Organization{
		NetworkID: 1,
		ID: -1,
		Name: "ordererorg",
		MSPID: "ordererMSP",
	}
	mspClient, err := org.NewMspClient()
	if err != nil {
		global.Logger.Error(fmt.Sprintf("fail to get %s mspClient", org.Name), zap.Error(err))
	}

	if err := mspClient.Enroll(username, mspclient.WithSecret(password)); err != nil {
		global.Logger.Error("fail to enroll user " + username, zap.Error(err))
	}

	sign, err := mspClient.GetSigningIdentity(fmt.Sprintf("Admin1@net%d.com", 1))
	if err != nil {
		global.Logger.Error(fmt.Sprintf("fail to get org%d AdminSigningIdentity", org.ID))
	}


	fmt.Println(sign.Identifier())
}

func TestUpdateNets(t *testing.T) {
	global.Nets["net1"] = net
	user := model.NewCaUserFromDomainName("User1@org1.net1.com")
	admin := model.NewCaUserFromDomainName("Admin1@net1.com")
	peer := model.NewCaUserFromDomainName("peer1.org1.net1.com")
	orderer := model.NewCaUserFromDomainName("orderer1.net1.com")

	channel := model.Channel{
		ID: 1,
		NetworkID: 1,
	}
	model.UpdateNets(user)
	n := global.Nets["net1"].(model.Network)
	n.Show()

	model.UpdateNets(admin)
	n = global.Nets["net1"].(model.Network)
	n.Show()

	model.UpdateNets(peer)
	n = global.Nets["net1"].(model.Network)
	n.Show()

	model.UpdateNets(orderer)
	n = global.Nets["net1"].(model.Network)
	n.Show()


	model.UpdateNets(channel)
	n = global.Nets["net1"].(model.Network)
	n.Show()
}

func TestGetAllAdminSign(t *testing.T) {
	model.UpdateNets(net)
	ids, err := net.GetAllAdminSigningIdentities()
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, id := range ids {
		fmt.Println(string(id.EnrollmentCertificate()))
	}
}

func TestGetPeerChannels(t *testing.T) {
	model.UpdateNets(net)

	ans, err := net.Organizations[1].Peers[0].GetJoinedChannel()
	if err != nil {
		global.Logger.Error("fail to get channels", zap.Error(err))
	}
	for _, s := range ans {
		fmt.Println(s)
	}
}

func TestGetCaCert(t *testing.T) {
	fmt.Println(model.NewCaUserFromDomainName("peer1.org1.net1.com").GetCACert())
}