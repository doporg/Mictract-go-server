package model

import (
	"database/sql/driver"
	"mictract/global"
	"strings"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"
)

type Organization struct {
	Name        string `json:"name"`
	Peers       Peers  `json:"peers"`
	MSPID       string `json:"mspid"`
	MSPPath     string `json:"msppath"`
	CAID        string `json:"caid"`
	NetworkName string `json:"networkname"`
}

type Organizations []Organization

// 自定义数据字段所需实现的两个接口
func (orgs *Organizations) Scan(value interface{}) error {
	return scan(&orgs, value)
}

func (orgs *Organizations) Value() (driver.Value, error) {
	return value(orgs)
}

func (org *Organization) GetMSPPath() string {
	return org.MSPPath
}

func (org *Organization) GetConfigtxFile() string {
	var configtxTemplate = `
Organizations:
    - &<OrgName>
        Name: <MSPID>
        ID: <MSPID>
        MSPDir: <MSPPath>
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('<MSPID>.admin', '<MSPID>.peer', '<MSPID>.client')"
            Writers:
                Type: Signature
                Rule: "OR('<MSPID>.admin', '<MSPID>.client')"
            Admins:
                Type: Signature
                Rule: "OR('<MSPID>.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('<MSPID>.peer')"
`
	strings.ReplaceAll(configtxTemplate, "<OrgName>", org.Name)
	strings.ReplaceAll(configtxTemplate, "<MSPID>", org.MSPID)
	return configtxTemplate
}

func (org *Organization) NewMspClient() (*mspclient.Client, error) {
	sdk, ok := global.SDKs[org.NetworkName]
	if !ok {
		return nil, errors.New("fail to get sdk. please update global.SDKs.")
	}
	return mspclient.New(sdk.Context(), mspclient.WithCAInstance(org.CAID), mspclient.WithOrg(org.Name))
}
