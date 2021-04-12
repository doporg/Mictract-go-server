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
)

// POST /organization
func AddOrg(c *gin.Context) {
	var info request.AddOrgReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.PeerCount < 1 {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage("An organization contains at least one peer").
			Result(c.JSON)
		return
	}

	net, err := dao.FindNetworkByID(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	netSvc := service.NewNetworkService(net)

	go func(){
		var newOrg *model.Organization
		if newOrg, err = netSvc.AddOrg(info.Nickname); err != nil {
			global.Logger.Error("fail to add org", zap.Error(err))
			dao.UpdateOrganizationStatusByID(newOrg.ID, enum.StatusError)
			return
		}
		orgSvc := service.NewOrganizationService(newOrg)

		// add rest peer
		for i := 1; i < info.PeerCount; i++ {
			if _, err := orgSvc.AddPeer(); err != nil {
				global.Logger.Error("fail to add rest peer", zap.Error(err))
				dao.UpdateOrganizationStatusByID(newOrg.ID, enum.StatusError)
				return
			}
		}
		dao.UpdateOrganizationStatusByID(newOrg.ID, enum.StatusRunning)
		global.Logger.Info("org has been created successfully!", zap.String("orgName", newOrg.GetName()))
		return
	}()

	response.Ok().
		Result(c.JSON)
}

// GET /api/organization
func ListOrganizations(c *gin.Context) {
	info := struct {
		NetworkID int `form:"networkID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.NetworkID == 0 {
		orgs, err := dao.FindAllOrganizations()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}
		response.Ok().
			SetPayload(respFactory.NewOrgs(orgs)).
			Result(c.JSON)

	} else {
		orgs, err := dao.FindAllOrganizationsInNetwork(info.NetworkID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}

		response.Ok().
			SetPayload(respFactory.NewOrgs(orgs)).
			Result(c.JSON)
	}
}