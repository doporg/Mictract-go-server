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

	net, err := model.FindNetworkByID(info.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func(){
		var newOrg *model.Organization
		if newOrg, err = net.AddOrg(info.Nickname); err != nil {
			global.Logger.Error("fail to add org", zap.Error(err))
			newOrg.UpdateStatus(enum.StatusError)
			return
		}

		// add rest peer
		for i := 1; i < info.PeerCount; i++ {
			if _, err := newOrg.AddPeer(); err != nil {
				global.Logger.Error("fail to add rest peer", zap.Error(err))
				newOrg.UpdateStatus(enum.StatusError)
				return
			}
		}
		newOrg.UpdateStatus(enum.StatusRunning)
		global.Logger.Error("org has been created successfully!", zap.String("orgName", newOrg.GetName()))
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
		nets, err := model.FindAllNetworks()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}
		orgs := []response.Organization{}
		for _, net := range nets {
			_orgs, err := net.GetOrganizations()
			if err != nil {
				response.Err(http.StatusInternalServerError, enum.CodeErrDB).
					SetMessage(err.Error()).
					Result(c.JSON)
			}
			orgs = append(orgs, response.NewOrgs(_orgs)...)
		}
		response.Ok().
			SetPayload(orgs).
			Result(c.JSON)

	} else {
		net, err := model.FindNetworkByID(info.NetworkID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		orgs, err := net.GetOrganizations()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
		}

		response.Ok().
			SetPayload(response.NewOrgs(orgs)).
			Result(c.JSON)
	}
}