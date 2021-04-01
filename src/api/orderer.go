package api

import (
	"github.com/gin-gonic/gin"
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
)

// POST /orderer/
func AddOrderer(c *gin.Context) {
	var info request.AddOrdererReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(model.NewCaUserFromDomainName(info.NetworkUrl).NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for i := 0; i < info.OrdererCount; i++ {
		if err := net.AddOrderersToSystemChannel(); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}

	net, _ = model.GetNetworkfromNets(net.ID)
	response.Ok().
		SetPayload(net).
		Result(c.JSON)
}

// GET /api/orderer
func ListOrderersByNetwork(c *gin.Context) {
	info := struct {
		NetworkURL string `form:"networkUrl" json:"networkUrl"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(model.NewCaUserFromDomainName(info.NetworkURL).NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewOrderers(net.Orders)).
		Result(c.JSON)
}
