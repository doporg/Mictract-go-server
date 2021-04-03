package response

import (
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
)

type Organization struct {
	Nickname		string		`json:"nickname"`
	OrganizationID 	int 		`json:"organizationID"`
	NetworkID 		int 		`json:"networkID"`
	Peers 			[]string 	`json:"peers"`
	Users 			[]string 	`json:"users"`
	Status 			string 		`json:"status"`
}

func NewOrg(o model.Organization) *Organization {
	ret := &Organization{
		Peers: []string{},
		Users: []string{},
		NetworkID: o.NetworkID,
		Status: o.Status,
		Nickname: o.Nickname,
	}

	// peers
	peers, err := o.GetPeers()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, peer := range peers {
		ret.Peers = append(ret.Peers, peer.GetName())
	}

	// users
	users, err := o.GetUsers()
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, user := range users {
		ret.Users = append(ret.Users, user.GetName())
	}

	return ret
}

func NewOrgs(os []model.Organization) []Organization {
	ret := []Organization{}
	for _, o :=range os {
		ret = append(ret, *NewOrg(o))
	}
	return ret
}