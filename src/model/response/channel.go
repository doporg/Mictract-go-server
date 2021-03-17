package response

import (
	"fmt"
	"mictract/model"
)

type Channel struct {
	Name string `json:"name"`
	Organizations []string `json:"organizations"`
	Status 	string `json:"status"`
}

func NewChannel(c model.Channel) Channel {
	ret := Channel{
		Name: fmt.Sprintf("channel%d", c.ID),
		Organizations: []string{},
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
