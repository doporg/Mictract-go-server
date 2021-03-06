package model

import (
	"database/sql/driver"
	"encoding/json"
)

// 作为向区块链网络提交链码交易的中转
// txID 查询执行成功后设置
type Transaction struct {
	ID			uint64		`json:"id"`
	TxID		string 		`json:"txID"`
	Status  	string 		`json:"status"`
	Message 	string 		`json:"message"`

	UserID  	int 		`json:"userID"`
	ChaincodeID int 		`json:"chaincodeID"`
	PeerURLs	mystring	`json:"peerURLs"`
	Args 		mystring 	`json:"args"`
	// init query execute
	InvokeType	string 		`json:"invokeType"`
}

// gorm need
type mystring []string
func (arr mystring) Value() (driver.Value, error) {
	return json.Marshal(arr)
}
func (arr *mystring) Scan(data interface{}) error {
	return json.Unmarshal(data.([]byte), &arr)
}