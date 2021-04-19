package dao

import (
	"errors"
	"mictract/global"
	"mictract/model"
)

func InsertOrganization(org *model.Organization) error {
	return global.DB.Create(org).Error
}

func FindAllOrganizations() ([]model.Organization, error) {
	var orgs []model.Organization
	if err := global.DB.Find(&orgs).Error; err != nil {
		return []model.Organization{}, err
	}
	return orgs, nil
}

func FindOrganizationByID(orgID int) (*model.Organization, error) {
	var orgs []model.Organization
	if err := global.DB.Where("id = ?", orgID).Find(&orgs).Error; err != nil {
		return &model.Organization{}, err
	}
	if len(orgs) < 1 {
		return &model.Organization{}, errors.New("no such org")
	}
	return &orgs[0], nil
}

func FindSystemUserInOrganization(orgID int) (*model.CaUser, error) {
	var sysUsers []model.CaUser
	if err := global.DB.
		Where("nickname = ? and organization_id = ?", "system-user", orgID).
		Find(&sysUsers).Error; err != nil {
		return &model.CaUser{}, err
	}
	if len(sysUsers) == 0{
		return &model.CaUser{}, errors.New("system user not found")
	}
	return &sysUsers[0], nil
}

func FindAllPeersInOrganization(orgID int) ([]model.CaUser, error) {
	peers, err := FindCaUserInOrganization(orgID, "peer")
	if err != nil {
		return []model.CaUser{}, err
	}
	if len(peers) <= 0 {
		return []model.CaUser{}, errors.New("No peer in org")
	}
	return peers, nil
}

func FindUserAndAdminInOrganization(orgID int) ([]model.CaUser, error) {
	users1, err := FindCaUserInOrganization(orgID, "user")
	if err != nil {
		return []model.CaUser{}, err
	}
	users2, err := FindCaUserInOrganization(orgID, "admin")
	if err != nil {
		return []model.CaUser{}, err
	}
	return append(users1, users2...), nil
}

func UpdateOrganizationStatusByID(orgID int, status string) error {
	return global.DB.Model(&model.Organization{}).Where("id = ?", orgID).Update("status", status).Error
}

func DeleteAllOrganizationsInNetwork(netID int) error {
	return global.DB.Where("network_id = ?", netID).Delete(&model.Organization{}).Error
}