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
	for _, orgName := range info.Orgs {
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

	if err := net.AddChannel(orgIDs); err != nil {
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

// GET /api/channel

// Note: All channel
func GetChannelInfo(c *gin.Context) {
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
		SetPayload(ret).
		Result(c.JSON)
}