package api

import (
	"github.com/gin-gonic/gin"
	"mictract/dao"
	"mictract/enum"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"net/http"
)

// POST /peer
func AddPeer(c *gin.Context) {
	var info request.AddPeerReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	org, err := dao.FindOrganizationByID(info.OrganizationID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	orgSvc := service.NewOrganizationService(org)
	for i := 0; i < info.PeerCount; i++ {
		if _, err := orgSvc.AddPeer(); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}

	response.Ok().
		Result(c.JSON)
}

// GET /api/peer
func ListPeersByOrganization(c *gin.Context) {
	info := struct {
		OrganizationID int `form:"organizationID" json:"organizationID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	peers, err := dao.FindAllPeersInOrganization(info.OrganizationID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewPeers(peers)).
		Result(c.JSON)
}