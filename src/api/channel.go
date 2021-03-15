package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-protos-go/common"
	"go.uber.org/zap"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
)

// POST /channel
// param: AddChannelReq
func AddChannel(c *gin.Context) {
	var info request.AddChannelReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(info.NetID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := net.AddChannel(info.OrgIDs); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, _ = model.GetNetworkfromNets(net.ID)
	response.Ok().
		SetPayload(net).
		Result(c.JSON)
}

// GET /channel

func GetChannelInfo(c *gin.Context) {
	var req request.ChannelInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var ch *model.Channel
	var err error
	if req.ChannelID == -1 {
		ch, err = model.GetSystemChannel(req.NetID)
	} else {
		ch, err = model.GetChannelFromNets(req.ChannelID, req.NetID)
	}

	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	BCI, err := ch.GetChannelInfo()
	if err != nil {
		global.Logger.Error("fail to query block chain info ", zap.Error(err))
		return
	}

	info := struct {
		BCI *common.BlockchainInfo `json:"blockChainInfo"`
		Channel model.Channel 	`json:"channel"`
	}{
		BCI: BCI.BCI,
		Channel: *ch,
	}

	code := enum.CodeErrBlockchainNetworkError
	if BCI.Status == 200 {
		code = enum.CodeOk
	}
	response.Err(int(BCI.Status), code).SetPayload(info).Result(c.JSON)

}