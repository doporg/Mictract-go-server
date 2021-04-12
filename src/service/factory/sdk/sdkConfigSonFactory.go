package sdk

import (
	"fmt"
	mConfig "mictract/config"
	"mictract/dao"
	"mictract/model"
	"path/filepath"
)

type SDKConfigSonFactory struct {
}

func NewSDKConfigSonFactory() *SDKConfigSonFactory  {
	return &SDKConfigSonFactory{}
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigClient(org *model.Organization) *model.SDKConfigClient  {
	return &model.SDKConfigClient{
		Organization: org.GetName(),
		Logging: struct {
			Level string "yaml:\"level\""
		}{Level: mConfig.SDK_LEVEL},
		Cryptoconfig: struct {
			Path string "yaml:\"path\""
		}{Path: filepath.Join(mConfig.LOCAL_BASE_PATH, model.GetNetworkNameByID(org.NetworkID))},
	}
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigOrganization(org *model.Organization) *model.SDKConfigOrganization {
	peers, _ := dao.FindAllPeersInOrganization(org.ID)
	users, _ := dao.FindUserAndAdminInOrganization(org.ID)
	ret := &model.SDKConfigOrganization{
		Mspid:                  org.GetMSPID(),
		Peers:                  []string{},
		Users: 					map[string]*model.SDKConfigOrganizationUser{},
		CertificateAuthorities: []string{org.GetCAID()},
		CryptoPath: filepath.Join(
			mConfig.LOCAL_BASE_PATH,
			model.GetNetworkNameByID(org.NetworkID),
			"peerOrganizations",
			fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID),
			"users", "{username}", "msp"),
	}

	for _, peer := range peers {
		ret.Peers = append(ret.Peers, peer.GetName())
	}
	for _, user := range users {
		ret.Users[user.GetName()] = sdkCSF.NewSDKConfigOrganizationUser(&user)
	}

	return ret
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigOrganizationUser(user *model.CaUser) *model.SDKConfigOrganizationUser {
	cert, _ := dao.FindCertByUserID(user.ID, false)
	return &model.SDKConfigOrganizationUser{
		Key: model.SDKConfigPem{
			Pem: cert.PrivateKey,
		},
		Cert: model.SDKConfigPem{
			Pem: cert.Certification,
		},
	}
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigNode(user *model.CaUser) *model.SDKConfigNode {
	cacert, _ := dao.FindCACertByOrganizationID(user.OrganizationID)
	port := 7051
	if user.IsInOrdererOrg() {
		port = 7050
	}
	return &model.SDKConfigNode{
		URL: fmt.Sprintf("grpcs://%s:%d", user.GetURL(), port),
		TLSCACerts: struct {
			Pem string "yaml:\"pem\""
		}{
			Pem: cacert.Certification,
		},
	}
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigChannel(ch *model.Channel) *model.SDKConfigChannel {
	peers, _ := dao.FindAllPeersInChannel(ch)
	return sdkCSF.newSDKConfigChannelByPeers(peers)
}

func (sdkCSF *SDKConfigSonFactory)NewDefaultSDKConfigChannel(netID int) *model.SDKConfigChannel {
	peers, _ := dao.FindAllPeersInNetwork(netID)
	return sdkCSF.newSDKConfigChannelByPeers(peers)
}

func (sdkCSF *SDKConfigSonFactory)newSDKConfigChannelByPeers(peers []model.CaUser) *model.SDKConfigChannel {
	ret := &model.SDKConfigChannel{
		Peers: map[string]struct {
			EndorsingPeer  bool "yaml:\"endorsingPeer\""
			ChaincodeQuery bool "yaml:\"chaincodeQuery\""
			LedgerQuery    bool "yaml:\"ledgerQuery\""
			EventSource    bool "yaml:\"eventSource\""
		}{},
	}

	for _, peer := range peers {
		ret.Peers[peer.GetName()] = struct {
			EndorsingPeer  bool "yaml:\"endorsingPeer\""
			ChaincodeQuery bool "yaml:\"chaincodeQuery\""
			LedgerQuery    bool "yaml:\"ledgerQuery\""
			EventSource    bool "yaml:\"eventSource\""
		}{
			EndorsingPeer:  true,
			ChaincodeQuery: true,
			LedgerQuery:    true,
			EventSource:    true,
		}
	}

	return ret
}

func (sdkCSF *SDKConfigSonFactory)NewSDKConfigCA(org *model.Organization) *model.SDKConfigCA {
	cacert, _ := dao.FindCACertByOrganizationID(org.ID)
	return &model.SDKConfigCA{
		URL: org.GetCAURLInK8S(),
		TLSCACerts: struct {
			Pem []string "yaml:\"pem\""
		}{Pem: []string{cacert.Certification}},
		Registrar: struct {
			EnrollId     string "yaml:\"enrollId\""
			EnrollSecret string "yaml:\"enrollSecret\""
		}{
			EnrollId:     "admin",
			EnrollSecret: "adminpw",
		},
	}
}
