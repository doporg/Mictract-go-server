package service

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/service/factory/sdk"

	pb "github.com/hyperledger/fabric-protos-go/peer"
)

type TransactionService struct {
	tx 	*model.Transaction
}

func NewTransactionService(tx *model.Transaction) *TransactionService {
	return &TransactionService{
		tx: tx,
	}
}

// shell批准时指定--init-required，或者sdk批准时指定 InitRequired = true，
// 运行链码时都需要先初始化链码，用--isInit或者IsInit: true
func (txSvc *TransactionService)InitCC(channelClient *channel.Client) (channel.Response, error) {
	_args := [][]byte{}
	if len(txSvc.tx.Args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(txSvc.tx.Args) > 1 {
		_args = packArgs(txSvc.tx.Args[1:])
	}
	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: 	model.GetChaincodeNameByID(txSvc.tx.ChaincodeID),
			Fcn: 			txSvc.tx.Args[0],
			Args: 			_args,
			IsInit: 		true,
		},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(txSvc.tx.PeerURLs...),
	)
	if err != nil {
		return response, errors.WithMessage(err, "fail to init chaincode")
	}
	return response, err
}

// If you do not specify peerURLs,
// the program seems to automatically find peers that meet the policy to endorse.
// If specified,
// you must be responsible for satisfying the endorsement strategy
func (txSvc *TransactionService)ExecuteCC(channelClient *channel.Client) (channel.Response, error) {
	_args := [][]byte{}
	if len(txSvc.tx.Args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(txSvc.tx.Args) > 1 {
		_args = packArgs(txSvc.tx.Args[1:])
	}

	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: 	model.GetChaincodeNameByID(txSvc.tx.ChaincodeID),
			Fcn: 			txSvc.tx.Args[0],
			Args: 			_args,
			IsInit: 		false,
		},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(txSvc.tx.PeerURLs...),
	)
	if err != nil {
		return channel.Response{}, errors.WithMessage(err, "fail to execute chaincode！")
	}

	return response, err
}

// eg: QueryCC(cc, "mycc", []string{"Query", "a"}, "peer0.org1.example.com")
func (txSvc *TransactionService)QueryCC(channelClient *channel.Client) (channel.Response, error) {
	_args := [][]byte{}
	if len(txSvc.tx.Args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(txSvc.tx.Args) > 1 {
		_args = packArgs(txSvc.tx.Args[1:])
	}

	response, err := channelClient.Query(
		channel.Request{
			ChaincodeID: model.GetChaincodeNameByID(txSvc.tx.ChaincodeID),
			Fcn: txSvc.tx.Args[0],
			Args: _args,
		},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(txSvc.tx.PeerURLs...),
	)
	if err != nil {
		return channel.Response{}, errors.WithMessage(err, "fail to execute qeury！")
	}
	return response, nil
}

func (txSvc *TransactionService) GetTransactionInBlockchain() (*pb.ProcessedTransaction, error) {
	var cc  		*model.Chaincode
	var ch  		*model.Channel
	var chSvc 		*ChannelService
	var orgID 		int
	var adminUser 	*model.CaUser
	var lc 			*ledger.Client
	var err 		error

	cc, err = dao.FindChaincodeByID(txSvc.tx.ChaincodeID)
	if err != nil {
		goto PrintErrorAndReturn
	}
	ch, err = dao.FindChannelByID(cc.ChannelID)
	if err != nil {
		goto PrintErrorAndReturn
	}
	chSvc = NewChannelService(ch)
	orgID = chSvc.ch.OrganizationIDs[0]
	adminUser, err = dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		goto PrintErrorAndReturn
	}

	lc, err = sdk.NewSDKClientFactory().NewLedgerClient(adminUser, chSvc.ch)
	if err != nil {
		goto PrintErrorAndReturn
	}

	return lc.QueryTransaction(fab.TransactionID(txSvc.tx.TxID), ledger.WithTargetEndpoints(txSvc.tx.PeerURLs...))

	PrintErrorAndReturn:
		global.Logger.Error("", zap.Error(err))
		return &pb.ProcessedTransaction{}, err
}

