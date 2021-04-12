package model

type SDKConfig struct {
	Version 				string 								`yaml:"version"`
	Client 					*SDKConfigClient 					`yaml:"client"`
	Organizations 			map[string]*SDKConfigOrganization 	`yaml:"organizations"`
	Orderers 				map[string]*SDKConfigNode 			`yaml:"orderers"`
	Peers 					map[string]*SDKConfigNode 			`yaml:"peers"`
	Channels 				map[string]*SDKConfigChannel 		`yaml:"channels"`
	CertificateAuthorities 	map[string]*SDKConfigCA 			`yaml:"certificateAuthorities"`
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

type SDKConfigOrganization struct {
	Mspid                  string   								`yaml:"mspid"`
	CryptoPath             string   								`yaml:"cryptoPath"`
	Peers                  []string 								`yaml:"peers"`
	Users				   map[string]*SDKConfigOrganizationUser 	`yaml:"users"`
	CertificateAuthorities []string 								`yaml:"certificateAuthorities"`
}

type SDKConfigOrganizationUser struct{
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

type SDKConfigCA struct {
	URL        string `yaml:"url"`

	TLSCACerts struct {
		Pem []string `yaml:"pem"`
	} `yaml:"tlsCACerts"`

	Registrar struct {
		EnrollId     string `yaml:"enrollId"`
		EnrollSecret string `yaml:"enrollSecret"`
	} `yaml:"registrar"`
}
