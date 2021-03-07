package model

import (
	"fmt"
	"mictract/config"
	"path/filepath"
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

func NewSDKConfig(n *Network) *SDKConfig {
	sdkconfig := SDKConfig{
		Version: "1.0.0",
		Client: &SDKConfigClient{
			Organization: "org1",
			Logging: struct {
				Level string "yaml:\"level\""
			}{Level: "info"},
			Cryptoconfig: struct {
				Path string "yaml:\"path\""
			}{Path: filepath.Join(config.LOCAL_BASE_PATH, n.Name)},
		},
		Organizations:          map[string]*SDKConfigOrganizations{},
		Orderers:               map[string]*SDKConfigNode{},
		Peers:                  map[string]*SDKConfigNode{},
		Channels:               map[string]*SDKConfigChannel{},
		CertificateAuthorities: map[string]*SDKCAs{},
	}
	// organizations
	for _, org := range n.Organizations {
		sdkconfig.Organizations[org.Name] = &SDKConfigOrganizations{
			Mspid:                  org.Name + "MSP",
			CryptoPath: filepath.Join(config.LOCAL_BASE_PATH, n.Name, "peerOrganizations", fmt.Sprintf("org%d.net%d.com", org.ID, n.ID),
				"users", "{username}", "msp"),
			// CryptoPath:             "peerOrganizations/" + org.Name + "." + n.Name + ".com/users/{username}/msp",
			Peers:                  []string{},
			Users: map[string]*SDKConfigOrganizationsUsers{},
			CertificateAuthorities: []string{},
		}

		for _, peer := range org.Peers {
			sdkconfig.Organizations[org.Name].Peers = append(sdkconfig.Organizations[org.Name].Peers, peer.Name)
		}

		if org.ID != -1 {
			sdkconfig.Organizations[org.Name].CertificateAuthorities = append(sdkconfig.Organizations[org.Name].CertificateAuthorities, "ca."+org.Name+"."+n.Name+".com")
		} else {
			sdkconfig.Organizations[org.Name].CertificateAuthorities = append(sdkconfig.Organizations[org.Name].CertificateAuthorities, "ca."+n.Name+".com")
		}


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
	for _, orderer := range n.Orders {
		sdkconfig.Orderers[orderer.Name] = &SDKConfigNode{
			URL: orderer.getURL(),
			TLSCACerts: struct {
				Pem string "yaml:\"pem\""
			}{
				Pem: NewCaUserFromDomainName(orderer.Name).GetCACert(),
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

	// peers
	for _, org := range n.Organizations {
		if org.ID == -1 {
			continue
		}
		for _, peer := range org.Peers {
			sdkconfig.Peers[peer.Name] = &SDKConfigNode{
				URL: peer.GetURL(),
				TLSCACerts: struct {
					Pem string "yaml:\"pem\""
				}{
					Pem: NewCaUserFromDomainName(peer.Name).GetCACert(),
				},
			}
			//channels _default
			sdkconfig.Channels["_default"].Peers[peer.Name] = struct {
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
	for _, channel := range n.Channels {
		channel.Name = fmt.Sprintf("channel%d", channel.ID)
		sdkconfig.Channels[channel.Name] = &SDKConfigChannel{
			Peers: map[string]struct {
				EndorsingPeer  bool "yaml:\"endorsingPeer\""
				ChaincodeQuery bool "yaml:\"chaincodeQuery\""
				LedgerQuery    bool "yaml:\"ledgerQuery\""
				EventSource    bool "yaml:\"eventSource\""
			}{},
		}
		for _, org := range channel.Organizations {
			for _, peer := range org.Peers {
				sdkconfig.Channels[channel.Name].Peers[peer.Name] = struct {
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
	for _, org := range n.Organizations {
		url := fmt.Sprintf("https://ca-org%d-net%d:7054", org.ID, org.NetworkID)
		caName := fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
		if org.ID == -1 {
			url = fmt.Sprintf("https://ca-net%d:7054", org.NetworkID)
			caName = fmt.Sprintf("ca.net%d.com", org.NetworkID)
		}
		sdkconfig.CertificateAuthorities[caName] = &SDKCAs{
			URL: url,
			TLSCACerts: struct {
				Pem []string "yaml:\"pem\""
			}{Pem: []string{GetCACertByOrgIDAndNetID(org.ID, org.NetworkID)}},
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
