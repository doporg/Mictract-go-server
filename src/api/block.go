package api

import (
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-protos-go/common"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
	"mictract/model/response"
	"mictract/service"
	"net/http"
)

// GET /block
func ListBlocks(c *gin.Context)  {
	var info struct{
		ChannelID 	int 	`form:"channelID" json:"channelID" binding:"required"`
		Page		int 	`form:"page" binding:"required"`
		PageSize	int 	`form:"pageSize" binding:"required"`
	}
	var ch *model.Channel
	var chSvc *service.ChannelService
	var err error

	var blocks []*common.Block

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err = dao.FindChannelByID(info.ChannelID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if ch.Status != enum.StatusRunning {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("check channel status").
			Result(c.JSON)
		return
	}

	chSvc = service.NewChannelService(ch)

	bcInfoResp, err := chSvc.GetChannelInfo()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	down := uint64(info.PageSize * (info.Page - 1))
	up   := uint64(info.PageSize * info.Page)
	if up > bcInfoResp.BCI.Height {
		up = bcInfoResp.BCI.Height
	}

	for i := down; i < up; i++ {
		block, err := chSvc.GetBlock(i)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		blocks = append(blocks, block)
	}

	ret, err := response.ParseBlocks(blocks)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(ret).
		Result(c.JSON)
}
