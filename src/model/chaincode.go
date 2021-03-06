package model

import (
	"fmt"
	"mictract/config"
	"path/filepath"
)

// Local chaincode
type Chaincode struct {
	ID  			int		                    `json:"id" gorm:"primarykey"`
	Nickname		string	                    `json:"nickname"`
	Status 			string						`json:"status"`

	ChannelID	 	int							`json:"channel_id"`
	NetworkID	 	int							`json:"network_id"`

	Label	 	 	string						`json:"label"`
	PolicyStr    	string						`json:"policy"`
	Version  	 	string 						`json:"version"`
	Sequence 	 	int64  						`json:"sequence"`
	InitRequired 	bool 						`json:"init_required"`

	PackageID	 	string						`json:"package_id"`
}

func GetChaincodeNameByID(ccID int) string {
	return fmt.Sprintf("chaincode%d", ccID)
}
func (c *Chaincode) GetName() string {
	return GetChaincodeNameByID(c.ID)
}

func (c *Chaincode) GetAddress() string {
	return fmt.Sprintf(
		"cc%d-chan%d-net%d:9999",
		c.ID,
		c.ChannelID,
		c.NetworkID)
}

func (c *Chaincode)GetCCPath() string {
	return filepath.Join(
		config.LOCAL_CC_PATH,
		fmt.Sprintf("chaincode%d", c.ID))
}