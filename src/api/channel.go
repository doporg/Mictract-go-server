package api

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
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

	net, err = model.FindNetworkByID(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func() {
		var ch *model.Channel
		if ch, err = net.AddChannel(info.OrganizationIDs, info.Nickname); err != nil {
			ch.UpdateStatus(enum.StatusError)
			global.Logger.Error("fail to add channel", zap.Error(err))
			return
		}
		ch.UpdateStatus(enum.StatusRunning)
		global.Logger.Info("channel has been created successfully", zap.String("channelName", ch.GetName()))
	}()

	response.Ok().
		Result(c.JSON)
}

// GET /api/channel

// Note: All channel
func GetChannelInfo(c *gin.Context) {
	info := struct {
		NetworkID int `form:"networkID" json:"networkID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.NetworkID != 0 {
		net, err := model.FindNetworkByID(info.NetworkID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		chs, err := net.GetChannels()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		response.Ok().
			SetPayload(response.NewChannels(chs)).
			Result(c.JSON)
	} else {
		ret := []model.Channel{}
		nets, err := model.FindAllNetworks()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}

		for _, net := range nets {
			chs, err := net.GetChannels()
			if err != nil {
				response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
					SetMessage(err.Error()).
					Result(c.JSON)
			}
			ret = append(ret, chs...)
		}

		response.Ok().
			SetPayload(response.NewChannels(ret)).
			Result(c.JSON)
	}
}