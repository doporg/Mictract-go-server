package test

import (
	"fmt"
	"go.uber.org/zap"
	"mictract/config"
	"mictract/global"
	"mictract/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveCaUser(t *testing.T) {
	// 一orderer，一组织网络
	tests := []struct {
		Username string
	}{
		{Username: "Admin1@net1.com"},
		{Username: "Admin1@org1.net1.com"},
		{Username: "User1@net1.com"},
		{Username: "User1@org1.net1.com"},
		{Username: "orderer1.net1.com"},
		{Username: "peer1.org1.net1.com"},
	}

	for _, tc := range tests {
		caUser := model.NewCaUserFromDomainName(tc.Username)
		_ = caUser.BuildDir([]byte("CA cert test"), []byte("cert test"), []byte("priv key test"), true)
		_ = caUser.BuildDir([]byte("CA cert test"), []byte("cert test"), []byte("priv key test"), false)
	}
}

func TestGetUsername(t *testing.T) {
	tests := []struct {
		Username string
		CaUser   *model.CaUser
	}{
		{
			Username: "Admin1@net1.com",
			CaUser:   model.NewAdminCaUser(1, -1, 1, ""),
		},
		{
			Username: "Admin1@org1.net1.com",
			CaUser:   model.NewAdminCaUser(1, 1, 1, ""),
		},
		{
			Username: "User1@net1.com",
			CaUser:   model.NewUserCaUser(1, -1, 1, ""),
		},
		{
			Username: "User1@org1.net1.com",
			CaUser:   model.NewUserCaUser(1, 1, 1, ""),
		},
		{
			Username: "orderer1.net1.com",
			CaUser:   model.NewOrdererCaUser(1, 1, ""),
		},
		{
			Username: "peer1.org1.net1.com",
			CaUser:   model.NewPeerCaUser(1, 1, 1, ""),
		},
	}

	for _, tc := range tests {
		username := tc.CaUser.GetUsername()
		assert.Equal(t, tc.Username, username)
	}
}

func TestNewCaUserFromUsername(t *testing.T) {
	tests := []struct {
		Username string
		CaUser   *model.CaUser
	}{
		{
			Username: "Admin1@net1.com",
			CaUser:   model.NewAdminCaUser(1, -1, 1, ""),
		},
		{
			Username: "Admin1@org1.net1.com",
			CaUser:   model.NewAdminCaUser(1, 1, 1, ""),
		},
		{
			Username: "User1@net1.com",
			CaUser:   model.NewUserCaUser(1, -1, 1, ""),
		},
		{
			Username: "User1@org1.net1.com",
			CaUser:   model.NewUserCaUser(1, 1, 1, ""),
		},
		{
			Username: "orderer1.net1.com",
			CaUser:   model.NewOrdererCaUser(1, 1, ""),
		},
		{
			Username: "peer1.org1.net1.com",
			CaUser:   model.NewPeerCaUser(1, 1, 1, ""),
		},
	}

	for _, tc := range tests {
		caUser := model.NewCaUserFromDomainName(tc.Username)
		assert.Equal(t, tc.CaUser, caUser)
	}
}

func TestGetBasePath(t *testing.T) {
	tests := []struct {
		BasePath string
		CaUser   *model.CaUser
	}{
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/ordererOrganizations/net1.com/users/Admin1@net1.com",
			CaUser: model.NewAdminCaUser(1, -1, 1, ""),
		},
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/peerOrganizations/org1.net1.com/users/Admin1@org1.net1.com",
			CaUser: model.NewAdminCaUser(1, 1, 1, ""),
		},
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/ordererOrganizations/net1.com/users/User1@net1.com",
			CaUser: model.NewUserCaUser(1, -1, 1, ""),
		},
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/peerOrganizations/org1.net1.com/users/User1@org1.net1.com",
			CaUser: model.NewUserCaUser(1, 1, 1, ""),
		},
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/ordererOrganizations/net1.com/orderers/orderer1.net1.com",
			CaUser: model.NewOrdererCaUser(1, 1, ""),
		},
		{
			BasePath: config.LOCAL_BASE_PATH +
				"/net1/peerOrganizations/org1.net1.com/peers/peer1.org1.net1.com",
			CaUser: model.NewPeerCaUser(1, 1, 1, ""),
		},
	}

	for _, tc := range tests {
		basePath := tc.CaUser.GetBasePath()
		assert.Equal(t, tc.BasePath, basePath)
	}
}

func TestStroeOrgMsp(t *testing.T) {
	// 一orderer，一组织网络
	causers := []model.CaUser {
		model.CaUser{
			OrganizationID: -1,
			NetworkID: 1,
		},
		model.CaUser{
			OrganizationID: 1,
			NetworkID: 1,
		},
	}

	for _, causer := range causers {
		err := causer.GenerateOrgMsp()
		if err != nil {
			global.Logger.Error("fail to generate orgMsp :", zap.Error(err))
		} else {
			fmt.Println(causer.GetCACert())
		}
		//fmt.Println(causer.GetCACert())
	}
}