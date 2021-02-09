package model

import (
	"mictract/model/request"
	"fmt"
)

type Network struct {
	ID 				int 			`json:"id"`
	Name			string			`json:"name" binding:"required"`
	Orders			[]Order			`json:"orders" binding:"required"`
	Organizations	[]Organization	`json:"organizations" binding:"required"`
	Consensus		string 			`json:"consensus" binding:"required"`
	TlsEnabled 		bool 			`json:"tlsEnabled"`
}

var (
	// just demo
	networks = []Network {
		{
			Name: "net1",
			Orders: []Order {
				{
					Name: "order1",
					Port: 1080,
				},
			},
			Organizations: []Organization {
				{
					Name:  "org1",
					Peers: []Peer {
						{
							Name: "peer1",
							Port: 1080,
						},
					},
				},
			},
			Consensus: "solo",
			TlsEnabled: true,
		},
	}

)

func FindNetworks(pageInfo request.PageInfo) ([]Network, error) {
	// TODO
	// find all networks in the `/networks` directory
	start := pageInfo.PageSize * ( pageInfo.Page - 1 )
	end := pageInfo.PageSize * pageInfo.Page
	if end > len(networks) {
		end = len(networks)
	}

	return networks[start: end], nil
}

func FindNetworkByID(id int) (Network, error) {
	// TODO
	for _, n := range networks {
		if id == n.ID {
			return n, nil
		}
	}

	return Network{}, fmt.Errorf("network not found")
}

func DeleteNetworkByID(id int) error {
	// TODO
	return nil
}

func (n *Network) Deploy() {
	// TODO
	// generate fabric-ca configurations and send them to k8s
	// enroll admin and register users to generate MSPs
	// generate order system and genesis block
	// generate organizations configurations and send them to k8s
	// create channel
	// join all peers into the channel
	// set the anchor peers for each org
}