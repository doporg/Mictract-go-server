package dao

import (
	"fmt"
	"github.com/pkg/errors"
	"mictract/config"
	"mictract/global"
	"mictract/model"
	"os"
	"path/filepath"
)

func InsertChaincode(cc *model.Chaincode) error {
	return global.DB.Create(cc).Error
}

func FindChaincodeByID(ccID int) (*model.Chaincode, error) {
	var ccs []model.Chaincode
	if err := global.DB.Where("id = ?", ccID).Find(&ccs).Error; err != nil {
		return &model.Chaincode{}, err
	}
	if len(ccs) == 0 {
		return &model.Chaincode{}, errors.New(fmt.Sprintf("no such chaincode(id = %d)", ccID))
	}
	return &ccs[0], nil
}

func DeleteChaincodeByID(ccID int) error {
	if err := global.DB.Where("id = ?", ccID).Delete(&model.Chaincode{}).Error; err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(config.LOCAL_CC_PATH, fmt.Sprintf("chaincode%d", ccID))); err != nil {
		return err
	}
	return nil
}

func FindAllChaincodes() ([]model.Chaincode, error) {
	ccs := []model.Chaincode{}
	if err := global.DB.Find(&ccs).Error; err != nil {
		return []model.Chaincode{}, err
	}
	return ccs, nil
}

func UpdateChaincodeNickname(ccID int, newNickname string) error {
	if err := global.DB.Model(&model.Chaincode{}).
		Where("id = ?", ccID).
		Updates(model.Chaincode{Nickname: newNickname}).Error; err != nil {
		return errors.WithMessage(err, "Fail to update")
	}
	return nil
}

func UpdateChaincodeStatusByID(ccID int, status string) error {
	return global.DB.Model(&model.Chaincode{}).Where("id = ?", ccID).Update("status", status).Error
}

func UpdateChaincodePackageIDByID(ccID int, packageID string) error  {
	return global.DB.Model(&model.Chaincode{}).Where("id = ?", ccID).Update("package_id", packageID).Error
}

func DeleteAllChaincodesInNetwork(netID int) error {
	return global.DB.Where("network_id = ?", netID).Delete(&model.Chaincode{}).Error
}
