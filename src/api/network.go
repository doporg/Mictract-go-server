package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
	"strconv"
)

// Create a new network configuration.
// Note that the network name can not be duplicated.
//
// POST	/network
// param: AddBasicNetwork
func CreateNetwork(c *gin.Context) {
	var info request.AddNetwork

	// check if request model contains some required fields.
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	fmt.Println(info)

	if info.Consensus != "solo" && info.Consensus != "etcdraft" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The consensus protocol only supports solo and etcdraft").
			Result(c.JSON)
		return
	}

	for _, val := range info.Orgs {
		if val < 1 {
			response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
				SetMessage("Every organization (including ordererorg) contains at least one node").
				Result(c.JSON)
			return
		}
	}

	if len(info.Orgs) < 2 {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The network contains at least one orderer and one peer").
			Result(c.JSON)
		return
	}

	// TODO
	// check if the network name has existed.
	// check if the new network configuration could be saved.
	net := model.GetBasicNetwork(info.Consensus)
	if err := net.Deploy(); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for i := 2; i < len(info.Orgs); i++ {
		if err := net.AddOrg(); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}

	// add rest peer
	for j := 1; j < len(info.Orgs); j++ {
		for i := 0; i < info.Orgs[j] - 1; i++ {
			net, _ = model.GetNetworkfromNets(net.ID)
			if err := net.Organizations[j].AddPeer(); err != nil {
				response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
					SetMessage(err.Error()).
					Result(c.JSON)
				return
			}
		}
	}

	// add rest orderer
	for i := 1; i < info.Orgs[0]; i++ {
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

// GET	/network
// param: PageInfo
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

// POST /network/addOrg
func AddOrg(c *gin.Context) {
	var info request.AddOrgReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(info.NetID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := net.AddOrg(); err != nil {
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

// POST /network/addPeer
func AddPeer(c *gin.Context) {
	var info request.AddPeerReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(info.NetID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.OrgID <= 0 || info.OrgID > len(net.Organizations) {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(fmt.Sprintf("org%d does not exist", info.OrgID)).
			Result(c.JSON)
		return
	}

	for i := 0; i < info.Num; i++ {
		if err := net.Organizations[info.OrgID].AddPeer(); err != nil {
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

// POST /network/addOrderer
func AddOrderer(c *gin.Context) {
	var info request.AddOrdererReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(info.NetID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for i := 0; i < info.Num; i++ {
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

