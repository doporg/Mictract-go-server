package model

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"gopkg.in/yaml.v3"
	mConfig "mictract/config"
	"mictract/global"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
)

type SDKConfig struct {
	Version string `yaml:"version"`

	Client *SDKConfigClient `yaml:"client"`

	Organizations map[string]*SDKConfigOrganizations `yaml:"organizations"`

	Orderers map[string]*SDKConfigNode `yaml:"orderers"`

	Peers map[string]*SDKConfigNode `yaml:"peers"`

	Channels map[string]*SDKConfigChannel `yaml:"channels"`

	CertificateAuthorities map[string]*SDKCAs `yaml:"certificateAuthorities"`
}

type SDKConfigClient struct {
	Organization string `yaml:"organization"`
	Logging      struct {
		Level string `yaml:"level"`
	} `yaml:"logging"`
	Cryptoconfig struct {
		Path string `yaml:"path"`
	} `yaml:"cryptoconfig"`
}

type SDKConfigOrganizations struct {
	Mspid                  string   `yaml:"mspid"`
	CryptoPath             string   `yaml:"cryptoPath"`
	Peers                  []string `yaml:"peers"`
	Users				map[string]*SDKConfigOrganizationsUsers `yaml:"users"`
	CertificateAuthorities []string `yaml:"certificateAuthorities"`
}
type SDKConfigOrganizationsUsers struct{
	Key   SDKConfigPem `yaml:"key"`
	Cert  SDKConfigPem `yaml:"cert"`
}
type SDKConfigPem struct {
	Pem string `yaml:"pem"`
}
type SDKConfigNode struct {
	URL        string `yaml:"url"`
	TLSCACerts struct {
		Pem string `yaml:"pem"`
	} `yaml:"tlsCACerts"`
}

type SDKConfigChannel struct {
	Peers map[string]struct {
		EndorsingPeer  bool `yaml:"endorsingPeer"`
		ChaincodeQuery bool `yaml:"chaincodeQuery"`
		LedgerQuery    bool `yaml:"ledgerQuery"`
		EventSource    bool `yaml:"eventSource"`
	} `yaml:"peers"`
}

type SDKCAs struct {
	URL        string `yaml:"url"`
	TLSCACerts struct {
		Pem []string `yaml:"pem"`
	} `yaml:"tlsCACerts"`
	Registrar struct {
		EnrollId     string `yaml:"enrollId"`
		EnrollSecret string `yaml:"enrollSecret"`
	} `yaml:"registrar"`
}

func NewSDKConfig(netID int) *SDKConfig {
	n, _ := FindNetworkByID(netID)
	orgs, _ := n.GetOrganizations()
	ordOrg, _ := n.GetOrdererOrganization()
	orgs = append(orgs, *ordOrg)
	orderers, _ := n.GetOrderers()
	chs, _ := n.GetChannels()

	sdkconfig := SDKConfig{
		Version: "1.0.0",
		Client: &SDKConfigClient{
			Organization: orgs[0].GetName(),
			Logging: struct {
				Level string "yaml:\"level\""
			}{Level: mConfig.SDK_LEVEL},
			Cryptoconfig: struct {
				Path string "yaml:\"path\""
			}{Path: filepath.Join(mConfig.LOCAL_BASE_PATH, n.GetName())},
		},
		Organizations:          map[string]*SDKConfigOrganizations{},
		Orderers:               map[string]*SDKConfigNode{},
		Peers:                  map[string]*SDKConfigNode{},
		Channels:               map[string]*SDKConfigChannel{},
		CertificateAuthorities: map[string]*SDKCAs{},
	}



	// organizations
	for _, org := range orgs {
		sdkconfig.Organizations[org.GetName()] = &SDKConfigOrganizations{
			Mspid:                  org.GetMSPID(),
			CryptoPath: filepath.Join(
				mConfig.LOCAL_BASE_PATH,
				n.GetName(),
				"peerOrganizations",
				fmt.Sprintf("org%d.net%d.com", org.ID, n.ID),
				"users", "{username}", "msp"),
			// CryptoPath:             "peerOrganizations/" + org.Name + "." + n.Name + ".com/users/{username}/msp",
			Peers:                  []string{},
			Users: map[string]*SDKConfigOrganizationsUsers{},
			CertificateAuthorities: []string{},
		}

		peers, _ := org.GetPeers()

		for _, peer := range peers {
			sdkconfig.Organizations[org.GetName()].Peers =
				append(sdkconfig.Organizations[org.GetName()].Peers, peer.GetName())
		}

		sdkconfig.Organizations[org.GetName()].CertificateAuthorities =
			append(sdkconfig.Organizations[org.GetName()].CertificateAuthorities, org.GetCAID())

		// users
		users, _ := org.GetUsers()

		for _, user := range users {
			sdkconfig.Organizations[org.GetName()].Users[user.GetName()] = &SDKConfigOrganizationsUsers{
				Key: SDKConfigPem{
					Pem: user.GetPrivateKey(),
				},
				Cert: SDKConfigPem{
					Pem: user.GetCert(),
				},
			}
		}
	}
	/*
	sdkconfig.Organizations["ordererorg"] = &SDKConfigOrganizations{
		Mspid:                  "ordererMSP",
		CryptoPath: filepath.Join(config.LOCAL_BASE_PATH, n.Name, "ordererOrganizations", n.Name + ".com", "users", "{username}", "msp"),
		// CryptoPath:             "ordererOrganizations/" + n.Name + ".com/users/{username}/msp",
		Peers:                  nil,
		Users: map[string]*SDKConfigOrganizationsUsers{},
		CertificateAuthorities: []string{},
	}
	sdkconfig.Organizations["ordererorg"].CertificateAuthorities = append(sdkconfig.Organizations["ordererorg"].CertificateAuthorities, "ca."+n.Name+".com")
	// users
	for _, user := range org.Users {
		causer := NewCaUserFromDomainName(user)
		sdkconfig.Organizations[org.Name].Users[user] = &SDKConfigOrganizationsUsers{
			Key: SDKConfigPem{
				Pem: causer.GetPrivateKey(),
			},
			Cert: SDKConfigPem{
				Pem: causer.GetCert(),
			},
		}
	}
	*/
	// orderers
	for _, orderer := range orderers {
		sdkconfig.Orderers[orderer.GetName()] = &SDKConfigNode{
			URL: fmt.Sprintf("grpcs://%s:7050", orderer.GetURL()),
			TLSCACerts: struct {
				Pem string "yaml:\"pem\""
			}{
				Pem: orderer.GetCACert(),
			},
		}
	}

	//channels _default
	sdkconfig.Channels["_default"] = &SDKConfigChannel{
		Peers: map[string]struct {
			EndorsingPeer  bool "yaml:\"endorsingPeer\""
			ChaincodeQuery bool "yaml:\"chaincodeQuery\""
			LedgerQuery    bool "yaml:\"ledgerQuery\""
			EventSource    bool "yaml:\"eventSource\""
		}{},
	}
	// channel system-channel
	sdkconfig.Channels["system-channel"] = &SDKConfigChannel{
		Peers: map[string]struct {
			EndorsingPeer  bool "yaml:\"endorsingPeer\""
			ChaincodeQuery bool "yaml:\"chaincodeQuery\""
			LedgerQuery    bool "yaml:\"ledgerQuery\""
			EventSource    bool "yaml:\"eventSource\""
		}{},
	}

	// peers
	for _, org := range orgs {
		if org.IsOrdererOrganization() {
			continue
		}
		peers, _ := org.GetPeers()
		for _, peer := range peers {
			sdkconfig.Peers[peer.GetName()] = &SDKConfigNode{
				URL: fmt.Sprintf("grpcs://%s:7051", peer.GetURL()),
				TLSCACerts: struct {
					Pem string "yaml:\"pem\""
				}{
					Pem: peer.GetCACert(),
				},
			}
			//channels _default
			sdkconfig.Channels["_default"].Peers[peer.GetName()] = struct {
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
			//channels system-channel
			sdkconfig.Channels["system-channel"].Peers[peer.GetName()] = struct {
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
	}

	// Users


	// channels else
	for _, ch := range chs {
		sdkconfig.Channels[ch.GetName()] = &SDKConfigChannel{
			Peers: map[string]struct {
				EndorsingPeer  bool "yaml:\"endorsingPeer\""
				ChaincodeQuery bool "yaml:\"chaincodeQuery\""
				LedgerQuery    bool "yaml:\"ledgerQuery\""
				EventSource    bool "yaml:\"eventSource\""
			}{},
		}
		orgs, _ := ch.GetOrganizations()

		for _, org := range orgs {
			peers, _ := org.GetPeers()
			for _, peer := range peers {
				sdkconfig.Channels[ch.GetName()].Peers[peer.GetName()] = struct {
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
		}

	}

	// certificateAuthorities
	for _, org := range orgs {
		sdkconfig.CertificateAuthorities[org.GetCAID()] = &SDKCAs{
			URL: org.GetCAURLInK8S(),
			TLSCACerts: struct {
				Pem []string "yaml:\"pem\""
			}{Pem: []string{org.GetCACert()}},
			Registrar: struct {
				EnrollId     string "yaml:\"enrollId\""
				EnrollSecret string "yaml:\"enrollSecret\""
			}{
				EnrollId:     "admin",
				EnrollSecret: "adminpw",
			},
		}
	}
	return &sdkconfig
}

func GetSDKByNetworkID(id int) (*fabsdk.FabricSDK, error) {
	configObj := NewSDKConfig(id)

	sdkconfig, err := yaml.Marshal(configObj)
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}

	// global.Logger.Info(string(sdkconfig))
	// for debug
	f, _ := os.Create(filepath.Join(mConfig.LOCAL_BASE_PATH, fmt.Sprintf("net%d", id), "sdk-config.yaml"))
	_, _ = f.WriteString(string(sdkconfig))
	_ = f.Close()

	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}
	updateSDKByNetworkID(id, sdk)
	return sdk, nil
}

func updateSDKByNetworkID(id int, sdk *fabsdk.FabricSDK) {
	oldSDK, ok := global.SDKs[id]
	if ok {
		oldSDK.Close()
	}
	global.SDKs[id] = sdk
}
