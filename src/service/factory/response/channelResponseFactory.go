package response

import (
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/response"
)

func NewChannel(c *model.Channel) *response.Channel {
	ret := &response.Channel{
		ChannelID: 		c.ID,
		Nickname: 		c.Nickname,
		Organizations: 	[]string{},
		NetworkID: 		c.NetworkID,
		Status: 		c.Status,
	}

	orgs, err := dao.FindAllOrganizationsInChannel(c)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, org := range orgs {
		ret.Organizations = append(ret.Organizations, org.GetName())
	}
	return ret
}

func NewChannels(cs []model.Channel) []response.Channel {
	ret := []response.Channel{}
	for _, c := range cs {
		ret = append(ret, *NewChannel(&c))
	}
	return ret
}
