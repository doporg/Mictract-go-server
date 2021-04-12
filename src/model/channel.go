package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Channel struct {
	ID            		int 				`json:"id"`
	Nickname	  		string 				`json:"nickname"`
	NetworkID     		int        			`json:"networkID"`
	Status 		  		string 				`json:"status"`

	OrganizationIDs		ints 				`json:"organization_ids"`
	OrdererIDs          ints				`json:"orderer_ids"`
}

// gorm need
type ints []int
func (arr ints) Value() (driver.Value, error) {
	return json.Marshal(arr)
}
func (arr *ints) Scan(data interface{}) error {
	return json.Unmarshal(data.([]byte), &arr)
}

func GetChannelNameByID(chID int) string {
	if chID == -1 {
		return "system-channel"
	}
	return fmt.Sprintf("channel%d", chID)
}
func (c *Channel) GetName() string {
	return GetChannelNameByID(c.ID)
}
