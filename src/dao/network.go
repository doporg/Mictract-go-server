package dao

import (
	"github.com/pkg/errors"
	"mictract/global"
	"mictract/model"
)

func InsertNetwork(net *model.Network) error {
	return global.DB.Create(net).Error
}

func FindAllNetworks() ([]model.Network, error){
	nets := []model.Network{}
	if err := global.DB.Find(&nets).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all nets")
	}
	return nets, nil
}

func FindNetworkByID(id int) (*model.Network, error) {
	var nets []model.Network
	if err := global.DB.Where("id = ?", id).Find(&nets).Error; err != nil {
		global.Logger.Error(err.Error())
		return &model.Network{}, err
	}
	if len(nets) == 0 {
		return &model.Network{}, errors.New("no such network")
	}
	return &nets[0], nil
}

func FindAllPeersInNetwork(netID int) ([]model.CaUser, error)  {
	cus := []model.CaUser{}
	if err := global.DB.Where("network_id = ? and type = ?", netID, "peer").Find(&cus).Error; err != nil {
		return []model.CaUser{}, err
	}
	if len(cus) == 0 {
		return []model.CaUser{}, errors.New("no peer in network")
	}
	return cus, nil
}

func FindAllOrderersInNetwork(netID int) ([]model.CaUser, error)  {
	cus := []model.CaUser{}
	if err := global.DB.Where("network_id = ? and type = ?", netID, "orderer").Find(&cus).Error; err != nil {
		return []model.CaUser{}, err
	}
	if len(cus) == 0 {
		return []model.CaUser{}, errors.New("no orderer in network")
	}
	return cus, nil
}

func FindAllOrganizationsInNetwork(netID int) ([]model.Organization, error) {
	orgs := []model.Organization{}
	if err := global.DB.Where("network_id = ?", netID).Find(&orgs).Error; err != nil {
		return []model.Organization{}, err
	}
	return orgs, nil
}

func FindOrdererOrganizationInNetwork(netID int) (*model.Organization, error) {
	orgs := []model.Organization{}
	if err := global.DB.Where("network_id = ? and is_orderer_org = ?", netID, true).Find(&orgs).Error; err != nil {
		return &model.Organization{}, err
	}
	return &orgs[0], nil
}

func FindAllChannelsInNetwork(netID int) ([]model.Channel, error) {
	chs := []model.Channel{}
	if err := global.DB.Where("network_id = ?", netID).Find(&chs).Error; err != nil {
		return []model.Channel{}, err
	}
	return chs, nil
}

func FindAllChaincodesInNetwork(netID int) ([]model.Chaincode, error) {
	ccs := []model.Chaincode{}
	if err := global.DB.Where("network_id = ?", netID).Find(&ccs).Error; err != nil {
		return []model.Chaincode{}, err
	}
	return ccs, nil
}

func FindAllUserInNetwork(netID int) ([]model.CaUser, error) {
	ccs := []model.CaUser{}
	if err := global.DB.Where("network_id = ? and type in ?", netID, []string{"user", "admin"}).Find(&ccs).Error; err != nil {
		return []model.CaUser{}, err
	}
	return ccs, nil
}

func UpdateNetworkStatusByID(id int, status string) error {
	return global.DB.Model(&model.Network{}).Where("id = ?", id).Update("status", status).Error
}

func DeleteNetworkByID(netID int) error {
	return global.DB.Where("id = ?", netID).Delete(&model.Network{}).Error
}