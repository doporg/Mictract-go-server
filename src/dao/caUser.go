package dao

import (
	"mictract/global"
	"mictract/model"
)

func InsertCaUser(cu *model.CaUser) error {
	return global.DB.Create(cu).Error
}

func FindCaUserInOrganization(orgID int, cuType string) ([]model.CaUser, error){
	var cus []model.CaUser
	if err := global.DB.
		Where("organization_id = ? and type = ?", orgID, cuType).Find(&cus).
		Error; err != nil {
		return []model.CaUser{}, err
	}
	return cus, nil
}

// user and admin
func FindCaUserInNetwork(netID int) ([]model.CaUser, error) {
	var cus []model.CaUser
	if err := global.DB.
		Where("network_id = ? and type in ?", netID, []string{"user", "admin"}).
		Find(&cus).Error; err != nil {
		return []model.CaUser{}, err
	}
	return cus, nil
}

// user and admin
func FindAllCaUser() ([]model.CaUser, error) {
	var cus []model.CaUser
	var err error
	if err = global.DB.Where("type in ?", []string{"user", "admin"}).Find(&cus).Error; err != nil {
		return []model.CaUser{}, err
	}
	return cus, nil
}

func FindCaUserByID(id int) (*model.CaUser, error) {
	var cus []model.CaUser
	if err := global.DB.Where("id = ?", id).Find(&cus).Error; err != nil {
		return &model.CaUser{}, err
	}
	return &cus[0], nil
}

func DeleteCaUserByID(caUserID int) error {
	return  global.DB.Where("id = ?", caUserID).Delete(&model.CaUser{}).Error
}

func DeleteAllCaUserInNetwork(netID int) error {
	return global.DB.Where("network_id = ?", netID).Delete(&model.CaUser{}).Error
}
