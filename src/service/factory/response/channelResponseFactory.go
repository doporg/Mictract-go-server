package response

import (
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/response"
)

func NewChannel(c *model.Channel) *response.Channel {
	orgs, err := dao.FindAllOrganizationsInChannel(c)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	return &response.Channel{
		ChannelID: 		c.ID,
		Nickname: 		c.Nickname,
		Organizations: 	NewOrgs(orgs),
		NetworkID: 		c.NetworkID,
		Status: 		c.Status,
	}
}

func NewChannels(cs []model.Channel) []response.Channel {
	ret := []response.Channel{}
	for _, c := range cs {
		ret = append(ret, *NewChannel(&c))
	}
	return ret
}
