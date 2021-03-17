package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
)

// GET /block
// Does not support system-channel
func GetBlockByBlockID(c *gin.Context) {
	var info request.BlockInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	fmt.Println(info)

	ch, err := model.GetChannelFromNets(info.ChannelID, info.NetID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ret, err := ch.GetBlock(info.BlockID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(ret).
		Result(c.JSON)
}
