package response

import (
	"mictract/model"
	"mictract/model/response"
)

func NewTransaction(tx *model.Transaction) *response.Transaction {
	return &response.Transaction{
		*tx,
	}
}

func NewTransactions(txs []model.Transaction) []response.Transaction {
	ret := []response.Transaction{}
	for _, tx := range txs {
		ret = append(ret, *NewTransaction(&tx))
	}
	return ret
}
