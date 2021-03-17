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
	var info request.AddNetworkReq

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

	if info.Consensus == "solo" && info.OrdererCount > 1 {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The solo consensus only supports one orderer").
			Result(c.JSON)
		return
	}

	if info.OrdererCount < 1 {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("Every organization (including ordererorg) contains at least one node").
			Result(c.JSON)
		return
	}
	for _, val := range info.PeerCounts {
		if val < 1 {
			response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
				SetMessage("Every organization (including ordererorg) contains at least one node").
				Result(c.JSON)
			return
		}
	}

	if len(info.PeerCounts) < 1 {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The network contains at least one org").
			Result(c.JSON)
		return
	}

	// TODO
	// check if the network name has existed.
	// check if the new network configuration could be saved.
	go func() {
		net := model.GetBasicNetwork(info.Consensus)
		if err := net.Deploy(); err != nil {
			net, _ = model.GetNetworkfromNets(net.ID)
			net.Status = "error"
			model.UpdateNets(*net)
			return
		}

		// add rest org
		for i := 1; i < len(info.PeerCounts); i++ {
			if err := net.AddOrg(); err != nil {
				net, _ = model.GetNetworkfromNets(net.ID)
				net.Status = "error"
				model.UpdateNets(*net)
				return
			}
		}

		// add rest peer
		for j := 0; j < len(info.PeerCounts); j++ {
			for i := 0; i < info.PeerCounts[j]-1; i++ {
				net, _ = model.GetNetworkfromNets(net.ID)
				if err := net.Organizations[j+1].AddPeer(); err != nil {
					net, _ = model.GetNetworkfromNets(net.ID)
					net.Status = "error"
					model.UpdateNets(*net)
					return
				}
			}
		}

		// add rest orderer
		for i := 1; i < info.OrdererCount; i++ {
			if err := net.AddOrderersToSystemChannel(); err != nil {
				net, _ = model.GetNetworkfromNets(net.ID)
				net.Status = "error"
				model.UpdateNets(*net)
				return
			}
		}
		net, _ = model.GetNetworkfromNets(net.ID)
		net.Status = "running"
		model.UpdateNets(*net)
	}()

	response.Ok().
		Result(c.JSON)
}

// GET	/network
// param: PageInfo
func ListNetworks(c *gin.Context) {
	//var pageInfo request.PageInfo
	//if err := c.ShouldBindQuery(&pageInfo); err != nil {
	//	response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
	//		SetMessage(err.Error()).
	//		Result(c.JSON)
	//	return
	//}
	if nets, err := model.QueryAllNetwork(); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			SetPayload(response.NewNetworks(nets)).
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
			SetPayload(response.NewNetwork(*net)).
			Result(c.JSON)
	}
}

// DELETE	/network/
func DeleteNetwork(c *gin.Context) {
	// id, err := strconv.Atoi(c.Param("id"))
	req := struct {
		URL string `form:"url" json:"url" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	id := model.NewCaUserFromDomainName(req.URL).NetworkID

	if err := model.DeleteNetworkByID(id); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			Result(c.JSON)
	}
}
