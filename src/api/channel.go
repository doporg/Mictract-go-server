package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	respFactory "mictract/service/factory/response"
	"mictract/service"
	"net/http"
	"strconv"
)

// POST /api/channel
// param: AddChannelReq
func AddChannel(c *gin.Context) {
	var info request.AddChannelReq
	var net *model.Network
	var err error

	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err = dao.FindNetworkByID(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func() {
		var ch *model.Channel
		if ch, err = service.NewNetworkService(net).AddChannel(info.OrganizationIDs, info.Nickname); err != nil {
			dao.UpdateChannelStatusByID(ch.ID, enum.StatusError)
			global.Logger.Error("fail to add channel", zap.Error(err))
			return
		}
		dao.UpdateChannelStatusByID(ch.ID, enum.StatusRunning)
		global.Logger.Info("channel has been created successfully", zap.String("channelName", ch.GetName()))
	}()

	response.Ok().
		Result(c.JSON)
}

// GET /api/channel
func ListChannels(c *gin.Context) {
	info := struct {
		NetworkID int `form:"networkID" json:"networkID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var chs []model.Channel
	var err error
	if info.NetworkID != 0 {
		chs, err = dao.FindAllChannelsInNetwork(info.NetworkID)
	} else {
		chs, err = dao.FindAllChannelsInNetwork(info.NetworkID)
	}
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(respFactory.NewChannels(chs)).
		Result(c.JSON)
}

// GET /api/channel/:id
func GetChannelByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err := dao.FindChannelByID(id)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	bcInfoResp, err := service.NewChannelService(ch).GetChannelInfo()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return

	}

	response.Ok().
		SetPayload(respFactory.NewChannelWithHeight(ch, bcInfoResp.BCI.Height)).
		Result(c.JSON)

}