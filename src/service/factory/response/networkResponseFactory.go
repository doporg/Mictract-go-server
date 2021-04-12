package response

import (
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/response"
	"strconv"
)

func NewNetwork(n *model.Network) *response.Network {
	orderers, err := dao.FindAllOrderersInNetwork(n.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	chs, err := dao.FindAllChannelsInNetwork(n.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	orgs, err := dao.FindAllOrganizationsInNetwork(n.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	ret := &response.Network{
		NetworkID: n.ID,
		Nickname: n.Nickname,
		Consensus: n.Consensus,
		TlsEnabled: n.TlsEnabled,
		Status: n.Status,
		CreateTime: strconv.FormatInt(n.CreatedAt.Unix(), 10),
		Orderers: []string{},
		Organizations: NewOrgs(orgs),
		// TODO
		Users: []response.User{},
		Channels: NewChannels(chs),
	}
	for _, orderer := range orderers {
		ret.Orderers = append(ret.Orderers, orderer.GetName())
	}
	return ret
}

func NewNetworks(ns []model.Network) []response.Network {
	ret := []response.Network{}
	for _, n := range ns {
		ret = append(ret, *NewNetwork(&n))
	}
	return ret
}
