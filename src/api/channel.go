package api

import (
	"fmt"
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
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	netID := model.NewCaUserFromDomainName(info.NetworkName).NetworkID
	orgIDs := []int{}
	for _, orgName := range info.Organizations {
		orgUser := model.NewCaUserFromDomainName(orgName)
		if orgUser.NetworkID != netID {
			response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
				SetMessage(fmt.Sprintf("The %s is not in the %s", orgName, info.NetworkName)).
				Result(c.JSON)
			return
		}
		orgIDs = append(orgIDs, orgUser.OrganizationID)
	}

	net, err := model.GetNetworkfromNets(netID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	newChID := len(net.Channels) + 1

	go func() {
		if err := net.AddChannel(orgIDs, info.Nickname); err != nil {
			n, _ := model.GetNetworkfromNets(netID)
			if newChID <= len(n.Channels) {
				n.Channels[newChID - 1].Status = "error"
			}
			model.UpdateNets(*n)
			global.Logger.Error("fail to add channel", zap.Error(err))
			return
		}
		n, _ := model.GetNetworkfromNets(netID)
		n.Channels[newChID - 1].Status = "running"
		model.UpdateNets(*n)
	}()

	response.Ok().
		Result(c.JSON)
}

// GET /api/channel

// Note: All channel
func GetChannelInfo(c *gin.Context) {
	info := struct {
		NetworkUrl string `form:"networkUrl"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.NetworkUrl != "" {
		net, err := model.GetNetworkfromNets(model.NewCaUserFromDomainName(info.NetworkUrl).NetworkID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		net.RefreshChannels()
		response.Ok().
			SetPayload(response.NewChannels(net.Channels)).
			Result(c.JSON)
	} else {
		ret := []model.Channel{}
		nets, err := model.QueryAllNetwork()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}

		for _, net := range nets {
			if err := net.RefreshChannels(); err != nil {
				response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
					SetMessage(err.Error()).
					Result(c.JSON)
			}
			ret = append(ret, net.Channels...)
		}

		response.Ok().
			SetPayload(response.NewChannels(ret)).
			Result(c.JSON)
	}
}