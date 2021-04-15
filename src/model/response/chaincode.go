package response

import (
	"mictract/model"
)

type Chaincode struct {
	Nickname	 	string 	`json:"nickname"`

	ChaincodeID 	int 	`json:"id"`
	NetworkID 		int 	`json:"networkID"`
	ChannelID 		int 	`json:"channelID"`

	Status		 	string 	`json:"status"`

	Label	 	 	string	`json:"label"`
	Address 	 	string	`json:"address"`
	PolicyStr    	string	`json:"policy"`
	Version  	 	string 	`json:"version"`
	Sequence 	 	int64  	`json:"sequence"`
	InitRequired 	bool 	`json:"initRequired"`

	PackageID	 	string	`json:"packageID"`
}

func NewChaincode(cc *model.Chaincode) Chaincode {
	return Chaincode{
		Nickname: cc.Nickname,
		Status: cc.Status,
		ChaincodeID: cc.ID,
		NetworkID: cc.NetworkID,
		ChannelID: cc.ChannelID,

		Label: cc.Label,
		PackageID: cc.PackageID,
		Address: cc.GetAddress(),
		PolicyStr: cc.PolicyStr,
		Version: cc.Version,
		Sequence: cc.Sequence,
		InitRequired: cc.InitRequired,
	}
}

func NewChaincodes(ccs []model.Chaincode) []Chaincode {
	_ccs := []Chaincode{}
	for _, cci := range ccs {
		_ccs = append(_ccs, NewChaincode(&cci))
	}
	return _ccs
}