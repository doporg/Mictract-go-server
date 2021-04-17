package response

import (
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/model/response"
)

func NewOrg(o *model.Organization) *response.Organization {
	// peers
	peers, err := dao.FindAllPeersInOrganization(o.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	// users
	users, err := dao.FindUserAndAdminInOrganization(o.ID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}

	return &response.Organization{
		OrganizationID: o.ID,
		Peers: 			response.NewPeers(peers),
		Users: 			response.NewUsers(users),
		NetworkID: 		o.NetworkID,
		Status: 		o.Status,
		Nickname: 		o.Nickname,
	}
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
