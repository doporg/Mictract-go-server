package model

import (
	"database/sql/driver"
)

type Peer struct {
	Name string `json:"name"`
}

type Peers []Peer

// 自定义数据字段所需实现的两个接口
func (peers *Peers) Scan(value interface{}) error {
	return scan(&peers, value)
}

func (peers *Peers) Value() (driver.Value, error) {
	return value(peers)
}
