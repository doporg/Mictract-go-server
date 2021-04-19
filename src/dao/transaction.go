package dao

import (
	"fmt"
	"github.com/pkg/errors"
	"mictract/global"
	"mictract/model"
)

func InsertTransaction(tx *model.Transaction) error {
	return global.DB.Create(tx).Error
}

func FindAllTransaction() ([]model.Transaction, error) {
	var txs []model.Transaction
	var err error
	if err = global.DB.Find(&txs).Error; err != nil {
		return []model.Transaction{}, err
	}
	return txs, nil
}

func FindTransactionByID(id int) (*model.Transaction, error) {
	var txs []model.Transaction
	if err := global.DB.Where("id = ?", id).Find(&txs).Error; err != nil {
		return &model.Transaction{}, err
	}
	if len(txs) < 1 {
		return &model.Transaction{}, errors.New(fmt.Sprintf("no such tx(id = %d)", id))
	}
	return &txs[0], nil
}

func UpdateTransactionStatusAndMessageByID(id uint64, status string, message string) error {
	return global.DB.Model(&model.Transaction{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "message": message}).
		Error
}

func UpdateTxIDByID(id uint64, txID string) error {
	return global.DB.Model(&model.Transaction{}).
		Where("id = ?", id).
		Update("tx_id", txID).
		Error
}

func DeleteTransaction(ids []int) error {
	return  global.DB.Where("id in ?", ids).Delete(&model.Transaction{}).Error
}

func DeleteAllTransaction() error {
	return  global.DB.Where("1 = 1").Delete(&model.Transaction{}).Error
}