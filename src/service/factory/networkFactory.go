package factory

import (
	"github.com/pkg/errors"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
	"time"
)

type NetworkFactory struct {
}

func NewNetworkFactory() *NetworkFactory {
	return &NetworkFactory{}
}

func (nf *NetworkFactory)NewNetwork(nickname, consensus string) (*model.Network, error) {
	// 1. check
	if consensus != "solo" && consensus != "etcdraft" {
		return &model.Network{}, errors.New("only supports solo and etcdraft")
	}

	// 2. new
	net := &model.Network{
		Nickname: 	nickname,
		CreatedAt: 	time.Now(),
		Status: 	enum.StatusStarting,
		Consensus: 	consensus,
	}

	// 3. insert into db
	if err := dao.InsertNetwork(net); err != nil {
		return &model.Network{}, errors.WithMessage(err, "Unable to insert network")
	}
	return net, nil
}
