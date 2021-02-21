package model

import "database/sql/driver"

type Channel struct {
	Name          string        `json:"name"`
	Organizations Organizations `json:"organizations"`
}

type Channels []Channel

// 自定义数据字段所需实现的两个接口
func (channels *Channels) Scan(value interface{}) error {
	return scan(&channels, value)
}

func (channels *Channels) Value() (driver.Value, error) {
	return value(channels)
}
