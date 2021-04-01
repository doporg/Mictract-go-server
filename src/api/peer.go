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

// POST /peer
func AddPeer(c *gin.Context) {
	var info request.AddPeerReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgUser := model.NewCaUserFromDomainName(info.Organization)

	net, err := model.GetNetworkfromNets(orgUser.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if orgUser.OrganizationID <= 0 || orgUser.OrganizationID > len(net.Organizations) {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(fmt.Sprintf("org%d does not exist", orgUser.OrganizationID)).
			Result(c.JSON)
		return
	}

	for i := 0; i < info.PeerCount; i++ {
		if err := net.Organizations[orgUser.OrganizationID].AddPeer(); err != nil {
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

// GET /api/peer
func ListPeersByOrganization(c *gin.Context) {
	info := struct {
		Organization string `form:"organization" json:"organization"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgUser := model.NewCaUserFromDomainName(info.Organization)
	org, err := model.GetOrgFromNets(orgUser.OrganizationID, orgUser.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewPeers(org.Peers)).
		Result(c.JSON)
}