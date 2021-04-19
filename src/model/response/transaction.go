package response

import "mictract/model"

type Transaction struct {
	model.Transaction
	//Payload 	[]byte 		`json:"payload"`
	//Signature   []byte		`json:"signature"`
}
