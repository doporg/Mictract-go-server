package model

import (
	"database/sql/driver"
	"fmt"
	"strings"

	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"
)

type Organization struct {
	ID int `json:"id"`
	NetworkID	int	`json:"networkid"`
	Name        string `json:"name"`
	MSPID       string `json:"mspid"`

	Peers       Peers  `json:"peers"`
	Users		[]string `json:"users"`

	// for add org (configtx.yaml)
	MSPPath     string `json:"msppath"`
	//CAID        string `json:"caid"`
	//NetworkName string `json:"networkname"`
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
	sdk, err := GetSDKByNetWorkID(org.NetworkID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk")
	}
	caID := ""
	if org.ID == -1 {
		caID = fmt.Sprintf("ca.net%d.com", org.NetworkID)
	} else {
		caID = fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	}
	return mspclient.New(sdk.Context(), mspclient.WithCAInstance(caID), mspclient.WithOrg(org.Name))
}
