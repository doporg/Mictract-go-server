package model

import (
	"database/sql/driver"
)

type Organization struct {
	Name  string `json:"name"`
	Peers Peers  `json:"peers"`
}

type Organizations []Organization

// 自定义数据字段所需实现的两个接口
func (orgs *Organizations) Scan(value interface{}) error {
	return scan(&orgs, value)
}

func (orgs *Organizations) Value() (driver.Value, error) {
	return value(orgs)
}
