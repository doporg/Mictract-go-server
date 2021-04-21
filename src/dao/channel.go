package dao

import (
	"github.com/pkg/errors"
	"mictract/global"
	"mictract/model"
)

func InsertChannel(ch *model.Channel) error {
	return global.DB.Create(ch).Error
}

func FindChannelByID(chID int) (*model.Channel, error) {
	var chs []model.Channel
	if err := global.DB.Where("id = ?", chID).Find(&chs).Error; err != nil {
		return &model.Channel{}, err
	}
	if len(chs) == 0 {
		return &model.Channel{}, errors.New("channel not found")
	}
	return &chs[0], nil
}

func FindAllChannels() ([]model.Channel, error) {
	var chs []model.Channel
	if err := global.DB.Find(&chs).Error; err != nil {
		return []model.Channel{}, err
	}
	return chs, nil
}

func UpdateOrgIDs(chID, orgID int) error {
	// 加个互斥锁
	global.ChannelLock.Lock()
	defer global.ChannelLock.Unlock()

	ch, err := FindChannelByID(chID)
	if err != nil {
		return err
	}

	ch.OrganizationIDs = append(ch.OrganizationIDs, orgID)

	return global.DB.Model(ch).Update("organization_ids", ch.OrganizationIDs).Error
}

func UpdateChannelStatusByID(chID int, status string) error {
	return global.DB.Model(&model.Channel{}).Where("id = ?", chID).Update("status", status).Error
}

func FindAllOrganizationsInChannel(c *model.Channel) ([]model.Organization, error) {
	if len(c.OrganizationIDs) <= 0 {
		return []model.Organization{}, errors.New("no organization in channel")
	}
	orgs := []model.Organization{}
	for _, orgID := range c.OrganizationIDs {
		org, err := FindOrganizationByID(orgID)
		if err != nil {
			return []model.Organization{}, err
		}
		orgs = append(orgs, *org)
	}
	return orgs, nil
}

func FindAllPeersInChannel(c *model.Channel) ([]model.CaUser, error) {
	peers := []model.CaUser{}
	for _, orgID := range c.OrganizationIDs {
		_peers, err := FindAllPeersInOrganization(orgID)
		if err != nil {
			return []model.CaUser{}, err
		}
		peers = append(peers, _peers...)
	}
	return peers, nil
}

func DeleteAllChannelsInNetwork(netID int) error {
	return global.DB.Where("network_id = ?", netID).Delete(&model.Channel{}).Error
}
