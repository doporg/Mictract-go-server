package response

import (
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/response"
)

func NewOrg(o *model.Organization) *response.Organization {
	ret := &response.Organization{
		OrganizationID: o.ID,
		Peers: 			[]string{},
		Users: 			[]string{},
		NetworkID: 		o.NetworkID,
		Status: 		o.Status,
		Nickname: 		o.Nickname,
	}

	// peers
	peers, err := dao.FindAllPeersInOrganization(o.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, peer := range peers {
		ret.Peers = append(ret.Peers, peer.GetName())
	}

	// users
	users, err := dao.FindUserAndAdminInOrganization(o.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	for _, user := range users {
		ret.Users = append(ret.Users, user.GetName())
	}

	return ret
}

func NewOrgs(os []model.Organization) []response.Organization {
	ret := []response.Organization{}
	for _, o :=range os {
		if o.IsOrdererOrganization() {
			continue
		}
		ret = append(ret, *NewOrg(&o))
	}
	return ret
}
