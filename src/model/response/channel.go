package response

import (
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
)

type Channel struct {
	ChannelID 		int 		`json:"channelID"`
	NetworkID		int 		`json:"networkID"`
	Nickname 		string 		`json:"nickname"`
	Organizations 	[]string 	`json:"organizations"`
	Status 			string 		`json:"status"`
}

func NewChannel(c model.Channel) *Channel {
	ret := &Channel{
		ChannelID: 		c.ID,
		Nickname: 		c.Nickname,
		Organizations: 	[]string{},
		NetworkID: 		c.NetworkID,
		Status: 		c.Status,
	}

	orgs, err := c.GetOrganizations()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, org := range orgs {
		ret.Organizations = append(ret.Organizations, org.GetName())
	}
	return ret
}

func NewChannels(cs []model.Channel) []Channel {
	ret := []Channel{}
	for _, c := range cs {
		ret = append(ret, *NewChannel(c))
	}
	return ret
}
