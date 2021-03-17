package response

import (
	"fmt"
	"mictract/model"
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
	ret := Network{
		Name: fmt.Sprintf("net%d", n.ID),
		Consensus: n.Consensus,
		TlsEnabled: n.TlsEnabled,
		Status: "running",
		CreateTime: n.CreatedAt.String(),
		Orderers: []string{},
		Organizations: NewOrgs(n.Organizations),
		// TODO
		Users: []User{},
		Channels: NewChannels(n.Channels),
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
