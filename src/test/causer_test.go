package test

import (
	"mictract/model"
	"mictract/utils"
	"testing"
)

func TestSave(t *testing.T) {
	// 一orderer，一组织网络
	causers := []model.CaUser{
		model.CaUser{
			Username: "orderer.net1.com",
		},
		model.CaUser{
			Username: "Admin@net1.com",
		},
		model.CaUser{
			Username: "peer1.org1.net1.com",
		},
		model.CaUser{
			Username: "Admin@org1.net1.com",
		},
		model.CaUser{
			Username: "User1@org1.net1.com",
		},
	}

	for _, causer := range causers {
		causer.Parse()
		utils.SaveCertAndPrivkey([]byte("CA cert test"), []byte("cert test"), []byte("priv key test"), false, causer)
		utils.SaveCertAndPrivkey([]byte("CA cert test"), []byte("cert test"), []byte("priv key test"), true, causer)
	}
}
