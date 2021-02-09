package api

import (
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// Create a new network configuration.
// Note that the network name can not be duplicated.
//
// POST	/network
// param: Network
func CreateNetwork(c *gin.Context) {
	var net model.Network

	// check if request model contains some required fields.
	if err := c.ShouldBindJSON(&net); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// TODO
	// check if the network name has existed.
	// check if the new network configuration could be saved.
	net.Deploy()

	response.Ok().
		SetPayload(net).
		Result(c.JSON)
}

// GET	/network
func ListNetworks(c *gin.Context) {
	var pageInfo request.PageInfo
	if err := c.ShouldBindJSON(&pageInfo); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if nets, err := model.FindNetworks(pageInfo); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			SetPayload(nets).
			Result(c.JSON)
	}
}

// GET	/network/:id
func GetNetwork(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if net, err := model.FindNetworkByID(id); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			SetPayload(net).
			Result(c.JSON)
	}
}

// DELETE	/network/:id
func DeleteNetwork(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := model.DeleteNetworkByID(id); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			Result(c.JSON)
	}
}
