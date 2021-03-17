package response

import (
	"fmt"
	"mictract/model"
	"strconv"
)

type Network struct {
	Name string `json:"name"`
	Consensus string `json:"consensus"`
	TlsEnabled bool `json:"tlsEnabled"`
	Status string `json:"status"`
	CreateTime string `json:"createTime"`
	Orderers []string `json:"orderers"`
	Organizations []Organization `json:"organizations"`
	Users []User `json:"users"`
	Channels []Channel `json:"channels"`
}

func NewNetwork(n model.Network) Network {
	orgs := []Organization{}
	if len(n.Organizations) >= 2 {
		orgs = NewOrgs(n.Organizations[1:])
	}
	ret := Network{
		Name: fmt.Sprintf("net%d", n.ID),
		Consensus: n.Consensus,
		TlsEnabled: n.TlsEnabled,
		Status: n.Status,
		CreateTime: strconv.FormatInt(n.CreatedAt.Unix(), 10),
		Orderers: []string{},
		Organizations: orgs,
		// TODO
		Users: []User{},
		Channels: NewChannels(n.Channels),
	}
	for _, orderer := range n.Orders {
		ret.Orderers = append(ret.Orderers, orderer.Name)
	}
	return ret
}

func NewNetworks(ns []model.Network) []Network {
	ret := []Network{}
	for _, n := range ns {
		ret = append(ret, NewNetwork(n))
	}
	return ret
}
