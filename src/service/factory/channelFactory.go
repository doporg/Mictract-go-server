package factory

import (
	"github.com/pkg/errors"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
)

type ChannelFactory struct {
}

func NewChannelFactory() *ChannelFactory {
	return &ChannelFactory{}
}

func (cf *ChannelFactory)NewChannel(netID int, nickname string, orgIDs []int) (*model.Channel, error) {
	// 1. check
	net, _ := dao.FindNetworkByID(netID)
	if net.Status != enum.StatusRunning {
		return &model.Channel{}, errors.New("Failed to call NewChannel, network status is abnormal")
	}

	if len(orgIDs) == 0 {
		return &model.Channel{}, errors.New("Failed to call NewChannel, orgIDs length is at least 1")
	}

	orderers, err := dao.FindAllOrderersInNetwork(netID)
	if err != nil {
		return &model.Channel{}, err
	}

	ch := &model.Channel{
		Nickname: 			nickname,
		NetworkID: 			netID,
		Status: 			enum.StatusStarting,
		OrganizationIDs: 	orgIDs,
		OrdererIDs: 		[]int{orderers[0].ID},
	}
	if err := dao.InsertChannel(ch); err != nil {
		return &model.Channel{}, err
	}
	return ch, nil
}

func (cf *ChannelFactory)NewSystemChannel(netID int) *model.Channel {
	return &model.Channel{
		ID: -1,
		NetworkID: netID,
		Status: enum.StatusRunning,
	}
}
