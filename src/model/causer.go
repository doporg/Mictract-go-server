package model

import "strings"

type CaUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`

	// user admin peer orderer
	MspType string
	OrgName string
	NetName string
	CAID    string
}

func (cu *CaUser) Parse() {
	if strings.Contains(cu.Username, "Admin") {
		cu.MspType = "admin"
	} else if strings.Contains(cu.Username, "@") {
		cu.MspType = "user"
	} else if strings.Contains(cu.Username, "org") {
		cu.MspType = "peer"
	} else {
		cu.MspType = "orderer"
	}
	tmp := strings.Split(cu.Username, ".")
	if cu.MspType == "orderer" {
		// orderer1.net2.com
		cu.OrgName = "ordererorg"
		cu.NetName = tmp[1]
	} else if cu.MspType == "peer" {
		// peer2.org3.net4.com
		cu.OrgName = tmp[1]
		cu.NetName = tmp[2]
	} else {
		if !strings.Contains(cu.Username, "org") {
			// Admin@net2.com User4@net3.com
			cu.OrgName = "ordererorg"
			cu.NetName = strings.Split(tmp[0], "@")[1]
		} else {
			// User5@org1.net5.com
			cu.OrgName = strings.Split(tmp[0], "@")[1]
			cu.NetName = tmp[1]
		}
	}
}
