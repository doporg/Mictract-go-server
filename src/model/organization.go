package model

import (
	"fmt"
	"github.com/pkg/errors"
	"mictract/config"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Organization struct {
	ID 					int 	`json:"id"`
	NetworkID			int		`json:"network_id"`
	Nickname			string 	`json:"nickname"`
	Status				string 	`json:"status"`

	CreatedAt 			time.Time
	IsOrdererOrg	 	bool
}

func GetOrganizationNameByIDAndBool(orgID int, isOrdOrg bool) string {
	if isOrdOrg {
		return  fmt.Sprintf("ordererorg")
	} else {
		return fmt.Sprintf("org%d", orgID)
	}
}

func GetOrganizationIDByName(orgName string) (int, error) {
	if len(orgName) < 4 {
		return -1, errors.New("invalid orgName")
	}
	return strconv.Atoi(orgName[3:])
}

func (org *Organization) IsOrdererOrganization() bool {
	return org.IsOrdererOrg
}

func (org *Organization) GetName() string {
	return GetOrganizationNameByIDAndBool(org.ID, org.IsOrdererOrganization())
}

func (org *Organization) GetMSPID() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("ordererMSP")
	} else {
		return fmt.Sprintf("org%dMSP", org.ID)
	}
}

func (org *Organization) GetCAID() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("ca.net%d.com", org.NetworkID)
	} else {
		return fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	}
}

func (org *Organization) GetCAURLInK8S() string {
	if org.IsOrdererOrganization() {
		return fmt.Sprintf("https://ca-net%d:7054", org.NetworkID)
	} else {
		return fmt.Sprintf("https://ca-org%d-net%d:7054", org.ID, org.NetworkID)
	}
}

func (org *Organization) GetMSPPath() string {
	ret := filepath.Join(config.LOCAL_BASE_PATH, fmt.Sprintf("net%d", org.NetworkID))
	if org.IsOrdererOrganization() {
		ret = filepath.Join(ret, "ordererOrganizations", fmt.Sprintf("net%d.com", org.NetworkID))
	} else {
		ret = filepath.Join(ret, "peerOrganizations", fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID))
	}
	return filepath.Join(ret, "msp")
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
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<OrgName>", org.GetName())
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPID>",   org.GetMSPID())
	configtxTemplate = strings.ReplaceAll(configtxTemplate, "<MSPPath>", org.GetMSPPath())
	return configtxTemplate
}


func (org *Organization) GetMSPDir() string {
	netName := fmt.Sprintf("net%d", org.NetworkID)

	basePath := filepath.Join(config.LOCAL_BASE_PATH, netName)

	if org.IsOrdererOrganization() {
		// ordererOrganizations
		basePath = filepath.Join(basePath, "ordererOrganizations", netName + ".com")
	} else {
		// peerOrganizations
		basePath = filepath.Join(basePath, "peerOrganizations",
			fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID))
	}

	// Build MSP directory by the given CaUser.
	return filepath.Join(basePath, "msp")
}
