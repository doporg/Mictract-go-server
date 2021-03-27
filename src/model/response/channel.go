package response

import (
	"fmt"
	"mictract/model"
)

type Channel struct {
	Name 			string 		`json:"name"`
	Nickname 		string 		`json:"nickname"`
	Organizations 	[]string 	`json:"organizations"`
	Network			string 		`json:"network"`
	Status 			string 		`json:"status"`
}

func NewChannel(c model.Channel) Channel {
	ret := Channel{
		Name: fmt.Sprintf("channel%d", c.ID),
		Nickname: c.Nickname,
		Organizations: []string{},
		Network: fmt.Sprintf("net%d.com", c.NetworkID),
		Status: c.Status,
	}
	for _, org := range c.Organizations {
		ret.Organizations = append(ret.Organizations, fmt.Sprintf("org%d.net%d.com", org.ID, org.NetworkID))
	}
	return ret
}

func NewChannels(cs []model.Channel) []Channel {
	ret := []Channel{}
	for _, c := range cs {
		ret = append(ret, NewChannel(c))
	}
	return ret
}
