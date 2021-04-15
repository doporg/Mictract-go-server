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
	users, err := dao.FindCaUserInNetwork(n.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	ret := &response.Network{
		ID: 			n.ID,
		Nickname: 		n.Nickname,
		Consensus: 		n.Consensus,
		TlsEnabled: 	n.TlsEnabled,
		Status: 		n.Status,
		CreateTime: 	strconv.FormatInt(n.CreatedAt.Unix(), 10),
		Orderers: 		response.NewOrderers(orderers),
		Organizations: 	NewOrgs(orgs),
		Users: 			response.NewUsers(users),
		Channels: 		NewChannels(chs),
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
