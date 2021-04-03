package api

import (
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
	var ch *model.Channel
	var err error

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err = model.FindChannelByID(info.ChannelID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.BlockID == -1 {
		ret, err := ch.GetChannelInfo()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		response.Ok().
			SetPayload(response.NewBlockHeightInfo(ret)).
			Result(c.JSON)
	} else {
		ret, err := ch.GetBlock(uint64(info.BlockID))
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
}
