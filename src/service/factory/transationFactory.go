package factory

import (
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
)

type TransationFactory struct {
}

func NewTransationFactory() *TransationFactory {
	return &TransationFactory{}
}

func (txf *TransationFactory) NewTransation(userID, chaincodeID int, peerURLs, args []string, invokeType string) (*model.Transaction, error) {
	tx := &model.Transaction{
		Status: 		enum.StatusExecute,
		UserID: 		userID,
		ChaincodeID: 	chaincodeID,
		PeerURLs: 		peerURLs,
		Args: 			args,
		InvokeType: 	invokeType,
	}
	if err := dao.InsertTransaction(tx); err != nil {
		return &model.Transaction{}, err
	}
	return tx, nil
}