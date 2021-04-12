package api

import (
	"fmt"
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
	"mictract/service/factory"
	"net/http"
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

	if len(info.PeerCounts) != len(info.OrgNicknames) {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("Organization information does not match").
			Result(c.JSON)
		return
	}

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

	// TODO
	// check if the network name has existed.
	// check if the new network configuration could be saved.
	go func() {
		net, err	:= factory.NewNetworkFactory().NewNetwork(info.Nickname, info.Consensus)
		if err != nil {
			global.Logger.Error("fail to init net", zap.Error(err))
			return
		}
		netSvc		:= service.NewNetworkService(net)

		if err = netSvc.Deploy(); err != nil {
			dao.UpdateNetworkStatusByID(net.ID, enum.StatusError)
			global.Logger.Error("fail to deploy basic network ", zap.Error(err))
			return
		}

		// add rest org
		for i := 0; i < len(info.PeerCounts); i++ {
			var newOrg *model.Organization
			if newOrg, err = netSvc.AddOrg(info.OrgNicknames[i]); err != nil {
				dao.UpdateNetworkStatusByID(net.ID, enum.StatusError)
				global.Logger.Error("fail to add rest org", zap.Error(err))
				return
			}
			// add rest peer
			for j := 0; j < info.PeerCounts[i] - 1; j++ {
				if _, err := service.NewOrganizationService(newOrg).AddPeer(); err != nil {
					dao.UpdateNetworkStatusByID(net.ID, enum.StatusError)
					global.Logger.Error("fail to add rest peer", zap.Error(err))
					return
				}
			}
			dao.UpdateOrganizationStatusByID(newOrg.ID, enum.StatusRunning)
		}


		// add rest orderer
		for i := 1; i < info.OrdererCount; i++ {
			if err := netSvc.AddOrderersToSystemChannel(); err != nil {
				dao.UpdateNetworkStatusByID(net.ID, enum.StatusError)
				global.Logger.Error("fail to add rest orderer", zap.Error(err))
				return
			}
		}
		dao.UpdateNetworkStatusByID(net.ID, enum.StatusRunning)
		global.Logger.Info("network has been created successfully", zap.String("netName", net.GetName()))
	}()

	response.Ok().
		Result(c.JSON)
}

// GET	/network
// param: PageInfo
func ListNetworks(c *gin.Context) {
	if nets, err := dao.FindAllNetworks(); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			SetPayload(respFactory.NewNetworks(nets)).
			Result(c.JSON)
	}
}

//// GET	/network/:id
//func GetNetwork(c *gin.Context) {
//	id, err := strconv.Atoi(c.Param("id"))
//
//	if err != nil {
//		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
//			SetMessage(err.Error()).
//			Result(c.JSON)
//		return
//	}
//
//	if net, err := model.FindNetworkByID(id); err != nil {
//		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
//			SetMessage(err.Error()).
//			Result(c.JSON)
//	} else {
//
//		response.Ok().
//			SetPayload(response.NewNetwork(*net)).
//			Result(c.JSON)
//	}
//}

// DELETE	/network/
func DeleteNetwork(c *gin.Context) {
	// id, err := strconv.Atoi(c.Param("id"))
	req := struct {
		NetworkID int `form:"networkID" json:"networkID" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := dao.FindNetworkByID(req.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := service.NewNetworkService(net).Delete(); err != nil {
		response.Err(http.StatusNotFound, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
	} else {
		response.Ok().
			Result(c.JSON)
	}
}
