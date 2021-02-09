package test

import (
	"mictract/model"
	"mictract/model/request"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testNet = model.Network {
	Name: "net1",
	Orders: []model.Order {
		{
			Name: "order1",
			Port: 1080,
		},
	},
	Organizations: []model.Organization {
		{
			Name:  "org1",
			Peers: []model.Peer {
				{
					Name: "peer1",
					Port: 1080,
				},
			},
		},
	},
	Consensus: "solo",
	TlsEnabled: true,
}

func TestCreateNetwork(t *testing.T) {
	tests := []struct {
		Net 	model.Network
		Code 	int
	} {
		{ Net: testNet,			Code: 200 },
		{ Net: model.Network{}, Code: 400 },
	}

	for _, tc := range tests {
		w := Post("/network/", Parse(tc.Net))

		fmt.Println(w.Body.String())
		assert.Equal(t, tc.Code, w.Code)
	}
}

func TestListNetworks(t *testing.T) {
	w := Get("/network/", Parse(request.PageInfo {
		Page:     1,
		PageSize: 2,
	}))

	fmt.Println(w.Body.String())
	assert.Equal(t, 200, w.Code)
}
