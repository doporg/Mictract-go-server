package api

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
	"net/http"
	"strconv"

	respFactory "mictract/service/factory/response"
)

// POST /api/transaction
func InvokeChaincode(c *gin.Context)  {
	var info request.InvokeCCReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.InvokeType != "init" &&
		info.InvokeType != "query" &&
		info.InvokeType != "execute" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("invokeType only supports init, execute, query").
			Result(c.JSON)
		return
	}

	cc, err := dao.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if cc.Status != enum.StatusRunning {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(fmt.Sprintf("the chaincode%d's status is %s", cc.ID, cc.Status)).
			Result(c.JSON)
		return
	}

	ch, err := dao.FindChannelByID(cc.ChannelID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	global.Logger.Info("Obtaining channel client...")
	user, err := dao.FindCaUserByID(info.UserID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	chClient, err := sdk.NewSDKClientFactory().NewChannelClientIncludeNetwork(user, ch)
	if err != nil {
		global.Logger.Error("fail to get channel client", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	tx, err := factory.NewTransationFactory().
		NewTransation(info.UserID, info.ChaincodeID, info.PeerURLs, info.Args, info.InvokeType)
	if err != nil {
		global.Logger.Error("fail to get new tx", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func(tx model.Transaction) {
		txSvc := service.NewTransactionService(&tx)
		var resp channel.Response
		switch info.InvokeType {
		case "init":
			resp, err = txSvc.InitCC(chClient,)
		case "query":
			resp, err = txSvc.QueryCC(chClient)
		case "execute":
			resp, err = txSvc.ExecuteCC(chClient)
		}
		if err != nil {
			global.Logger.Error(err.Error())
			dao.UpdateTransactionStatusAndMessageByID(
				tx.ID,
				enum.StatusError,
				base64.StdEncoding.EncodeToString([]byte(err.Error())),
			)
			return
		}
		global.Logger.Info(fmt.Sprintf("txID = %s", resp.TransactionID))
		if err := dao.UpdateTxIDByID(tx.ID, string(resp.TransactionID)); err != nil {
			dao.UpdateTransactionStatusAndMessageByID(
				tx.ID,
				enum.StatusError,
				fmt.Sprintf("fail to update txID(txID = %s)", resp.TransactionID),
			)
			return
		}
		dao.UpdateTransactionStatusAndMessageByID(tx.ID, enum.StatusSuccess, "well done")

	}(*tx)

	response.Ok().Result(c.JSON)
}

// GET /api/transaction
func ListTransaction(c *gin.Context)  {
	txs, err := dao.FindAllTransaction()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
	}
	response.Ok().SetPayload(respFactory.NewTransactions(txs)).Result(c.JSON)
}

// GET /api/transation/:id
func GetTransactionInBlockchain(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	tx, err := dao.FindTransactionByID(id)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if tx.Status != enum.StatusSuccess {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("Abnormal transaction status").
			Result(c.JSON)
		return
	}

	txSvc := service.NewTransactionService(tx)
	resp, err := txSvc.GetTransactionInBlockchain()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().SetPayload(resp).Result(c.JSON)
}

// DELETE /api/transaction
func DeleteTransaction(c *gin.Context)  {
	info := struct {
		IDs 	[]int `form:"ids" json:"ids" binding:"required"`
	}{}
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if len(info.IDs) == 0 {
		if err := dao.DeleteAllTransaction(); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	} else {
		if err := dao.DeleteTransaction(info.IDs); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}
	response.Ok().Result(c.JSON)
}