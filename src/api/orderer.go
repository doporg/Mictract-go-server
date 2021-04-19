package api

import (
	"github.com/gin-gonic/gin"
	"mictract/dao"
	"mictract/enum"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"net/http"
	"strconv"
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

	net, err := dao.FindNetworkByID(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	netSvc := service.NewNetworkService(net)

	for i := 0; i < info.OrdererCount; i++ {
		if err := netSvc.AddOrderersToSystemChannel(); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}

	response.Ok().
		Result(c.JSON)
}

// GET /api/orderer
func ListOrderersByNetwork(c *gin.Context) {
	info := struct {
		NetworkID int `form:"networkID" json:"networkID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orderers, err := dao.FindAllOrderersInNetwork(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewOrderers(orderers)).
		Result(c.JSON)
}

// GET /api/orderer
func GetOrdererByID(c *gin.Context)  {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orderer, err := dao.FindCaUserByID(id)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewOrderer(orderer)).
		Result(c.JSON)
}
