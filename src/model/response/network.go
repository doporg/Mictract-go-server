package response

import (
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
	"strconv"
)

type Network struct {
	NetworkID 		int 			`json:"networkID"`
	Nickname 		string 			`json:"nickname"`
	Consensus 		string 			`json:"consensus"`
	TlsEnabled 		bool 			`json:"tlsEnabled"`
	Status 			string 			`json:"status"`
	CreateTime 		string 			`json:"createTime"`
	Orderers 		[]string 		`json:"orderers"`
	Organizations 	[]Organization 	`json:"organizations"`
	Users 			[]User 			`json:"users"`
	Channels 		[]Channel		`json:"channels"`
}

func NewNetwork(n model.Network) *Network {
	orderers, err := n.GetOrderers()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	chs, err := n.GetChannels()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	orgs, err := n.GetOrganizations()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	ret := &Network{
		NetworkID: n.ID,
		Nickname: n.Nickname,
		Consensus: n.Consensus,
		TlsEnabled: n.TlsEnabled,
		Status: n.Status,
		CreateTime: strconv.FormatInt(n.CreatedAt.Unix(), 10),
		Orderers: []string{},
		Organizations: NewOrgs(orgs),
		// TODO
		Users: []User{},
		Channels: NewChannels(chs),
	}
	for _, orderer := range orderers {
		ret.Orderers = append(ret.Orderers, orderer.GetName())
	}
	return ret
}

func NewNetworks(ns []model.Network) []Network {
	ret := []Network{}
	for _, n := range ns {
		ret = append(ret, *NewNetwork(n))
	}
	return ret
}
