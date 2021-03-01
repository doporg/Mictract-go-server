package model

import (
	"database/sql/driver"
	"fmt"
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

func (orderer *Order)getURL() string {
	causer := NewCaUserFromDomainName(orderer.Name)
	return fmt.Sprintf("grpcs://orderer%d-org%d-net%d:7050", causer.UserID, causer.OrganizationID, causer.NetworkID)
}
