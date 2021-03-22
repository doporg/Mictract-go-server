package response

import (
	"fmt"
	"mictract/model"
)

type Organization struct {
	Name 		string 		`json:"name"`
	Network 	string 		`json:"network"`
	Peers 		[]string 	`json:"peers"`
	Users 		[]string 	`json:"users"`
	Status 		string 		`json:"status"`
}

func NewOrg(o model.Organization) Organization {
	ret := Organization{
		Peers: []string{},
		Users: []string{},
		Network: fmt.Sprintf("net%d.com", o.NetworkID),
		Status: o.Status,
	}

	// peers
	for _, peer := range o.Peers {
		ret.Peers = append(ret.Peers, peer.Name)
	}

	// users
	for _, user := range o.Users {
		ret.Users = append(ret.Users, user)
	}

	if o.ID == -1 {
		ret.Name = "ordererorg"
	} else {
		ret.Name = fmt.Sprintf("org%d.net%d.com", o.ID, o.NetworkID)
	}

	return ret
}

func NewOrgs(os []model.Organization) []Organization {
	ret := []Organization{}
	for _, o :=range os {
		ret = append(ret, NewOrg(o))
	}
	return ret
}