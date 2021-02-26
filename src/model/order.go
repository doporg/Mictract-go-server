package model

import (
	"database/sql/driver"
)

type Order struct {
	// Name should be domain name.
	// Example: orderer1.net1.com
	Name string `json:"name"`
}

type Orders []Order

// 自定义数据字段所需实现的两个接口
func (orderers *Orders) Scan(value interface{}) error {
	return scan(&orderers, value)
}

func (orderers *Orders) Value() (driver.Value, error) {
	return value(orderers)
}
