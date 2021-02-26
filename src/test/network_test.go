package test

import (
	"fmt"
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testNet = model.Network{
	Name: "net1",
	Orders: []model.Order{
		{
			Name: "order1.net1.com",
		},
	},
	Organizations: []model.Organization{
		{
			Name: "org1.net1.com",
			Peers: []model.Peer{
				{
					Name: "peer1.org1.net1.com",
				},
			},
		},
	},
	Consensus:  "solo",
	TlsEnabled: true,
}

func TestCreateNetwork(t *testing.T) {
	tests := []struct {
		Net  model.Network
		Code int
	}{
		{Net: testNet, Code: 200},
		{Net: model.Network{}, Code: 400},
	}

	for _, tc := range tests {
		w := Post("/network/", Parse(tc.Net))

		fmt.Println(w.Body.String())
		assert.Equal(t, tc.Code, w.Code)
	}
}

func TestListNetworks(t *testing.T) {
	w := Get("/network/", Parse(request.PageInfo{
		Page:     1,
		PageSize: 2,
	}))

	fmt.Println(w.Body.String())
	assert.Equal(t, 200, w.Code)
}

func TestDeployNetwork(t *testing.T) {
	var err error
	if err = testNet.Deploy(); err != nil {
		global.Logger.Error("testing", zap.Error(err))
	}

	assert.Equal(t, true, err == nil)
}