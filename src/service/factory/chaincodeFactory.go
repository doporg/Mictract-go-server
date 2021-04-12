package factory

import (
	"github.com/pkg/errors"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"os"
)

type ChaincodeFactory struct {

}

func NewChaincodeFactory() *ChaincodeFactory {
	return &ChaincodeFactory{}
}

// tar czf src.tar.gz src
func (ccf *ChaincodeFactory)NewChaincode(nickname string, chID, netID int, label, policyStr, version string,
	seq int64, initReq bool) (*model.Chaincode, error){
	// 1. check
	net, _ := dao.FindNetworkByID(netID)
	if net.Status != enum.StatusRunning {
		return &model.Chaincode{}, errors.New("Unable to create chaincode, please check network status")
	}
	ch, _ := dao.FindChannelByID(chID)
	if ch.Status != enum.StatusRunning {
		return &model.Chaincode{}, errors.New("Unable to create chaincode, please check channel status")
	}

	cc := &model.Chaincode{
		Nickname: nickname,
		Status: enum.StatusUnpacking,
		ChannelID: chID,
		NetworkID: netID,

		Label: label,
		PolicyStr: policyStr,
		Version: version,
		Sequence: seq,
		InitRequired: initReq,
	}

	if err := global.DB.Create(&cc).Error; err != nil {
		return cc, err
	}

	//cc.ID
	// mkdir chaincodes/chaincodeID
	if err := os.MkdirAll(cc.GetCCPath(), os.ModePerm); err != nil {
		return cc, err
	}

	return cc, nil
}