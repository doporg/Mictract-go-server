package model

import (
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
	CertificateAuthorities []string `yaml:"certificateAuthorities"`
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
			CryptoPath:             "peerOrganizations/" + org.Name + "." + n.Name + ".com/users/{username}/msp",
			Peers:                  []string{},
			CertificateAuthorities: []string{},
		}
		for _, peer := range org.Peers {
			sdkconfig.Organizations[org.Name].Peers = append(sdkconfig.Organizations[org.Name].Peers, peer.Name)
		}

		sdkconfig.Organizations[org.Name].CertificateAuthorities = append(sdkconfig.Organizations[org.Name].CertificateAuthorities, "ca."+org.Name+"."+n.Name+".com")
	}
	sdkconfig.Organizations["ordererorg"] = &SDKConfigOrganizations{
		Mspid:                  "ordererorg" + "MSP",
		CryptoPath:             "ordererOrganizations/" + n.Name + ".com/users/{username}/msp",
		Peers:                  nil,
		CertificateAuthorities: []string{},
	}
	sdkconfig.Organizations["ordererorg"].CertificateAuthorities = append(sdkconfig.Organizations["ordererorg"].CertificateAuthorities, "ca."+n.Name+".com")

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

	// channels else
	for _, channel := range n.Channels {
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
	sdkconfig.CertificateAuthorities["ca."+n.Name+".com"] = &SDKCAs{
		URL: "ca-" + n.Name,
		TLSCACerts: struct {
			Pem []string "yaml:\"pem\""
		}{Pem: []string{NewCaUserFromDomainName("orderer1." + n.Name + ".com").GetCACert()}},
		Registrar: struct {
			EnrollId     string "yaml:\"enrollId\""
			EnrollSecret string "yaml:\"enrollSecret\""
		}{
			EnrollId:     "admin",
			EnrollSecret: "adminpw",
		},
	}
	for _, org := range n.Organizations {
		sdkconfig.CertificateAuthorities["ca." + org.Name + "." + n.Name + ".com"] = &SDKCAs{
			URL: "https://ca-" + org.Name + "-" + n.Name + ":7054",
			TLSCACerts: struct {
				Pem []string "yaml:\"pem\""
			}{Pem: []string{NewCaUserFromDomainName(org.Peers[0].Name).GetCACert()}},
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
