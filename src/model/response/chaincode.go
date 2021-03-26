package response

import (
	"fmt"
	"go.uber.org/zap"
	"mictract/global"
	"mictract/model"
)

type Chaincode struct {
	Status		 string `json:"status"`
	CCID 		 int 	`json:"ccid"`
	Label	 	 string	`json:"label"`
	Address 	 string	`json:"address"`
	PolicyStr    string	`json:"policy"`
	Version  	 string `json:"version"`
	Sequence 	 int64  `json:"sequence"`
	InitRequired bool 	`json:"init_required"`

	PackageID	 string	`json:"package_id"`

	ChannelName	 string	`json:"channelName"`
	NetworkUrl	 string	`json:"networkUrl"`
}

func NewChaincode(cci *model.ChaincodeInstance) Chaincode {
	cc, err := model.GetChaincodeByID(cci.CCID)
	if err != nil {
		global.Logger.Error("fail to get cc ", zap.Error(err))
	}
	return Chaincode{
		Status: cc.Status,
		CCID: cc.ID,
		Label: cci.Label,
		PackageID: cci.PackageID,
		Address: cci.Address,
		PolicyStr: cci.PolicyStr,
		Version: cci.Version,
		Sequence: cci.Sequence,
		InitRequired: cci.InitRequired,
		ChannelName: fmt.Sprintf("channel%d", cci.ChannelID),
		NetworkUrl: fmt.Sprintf("net%d.com", cci.NetworkID),
	}
}

func NewChaincodes(ccis []model.ChaincodeInstance) []Chaincode {
	ccs := []Chaincode{}
	for _, cci := range ccis {
		ccs = append(ccs, NewChaincode(&cci))
	}
	return ccs
}