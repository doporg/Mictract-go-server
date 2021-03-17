package api

import (
	"github.com/gin-gonic/gin"
	"mictract/enum"
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

	netID := model.NewCaUserFromDomainName(info.NetworkUrl).NetworkID
	net, err := model.GetNetworkfromNets(netID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	newOrgID := len(net.Organizations)

	go func(){
		orgID := len(net.Organizations)
		if err := net.AddOrg(); err != nil {
			n, _ := model.GetNetworkfromNets(net.ID)
			if newOrgID < len(n.Organizations) {
				n.Organizations[newOrgID].Status = "error"
			}
			model.UpdateNets(*n)
			return
		}

		// add rest peer
		org, err := model.GetOrgFromNets(orgID, netID)
		if err != nil {
			n, _ := model.GetNetworkfromNets(net.ID)
			if newOrgID < len(n.Organizations) {
				n.Organizations[newOrgID].Status = "error"
			}
			model.UpdateNets(*n)
			return
		}
		for i := 1; i < info.PeerCount; i++ {
			if err := org.AddPeer(); err != nil {
				n, _ := model.GetNetworkfromNets(net.ID)
				if newOrgID < len(n.Organizations) {
					n.Organizations[newOrgID].Status = "error"
				}
				model.UpdateNets(*n)
				return
			}
		}
		n, _ := model.GetNetworkfromNets(net.ID)
		n.Organizations[newOrgID].Status = "success"
		model.UpdateNets(*n)
		return
	}()


	response.Ok().
		Result(c.JSON)

}

// GET /api/organization
func ListOrganizations(c *gin.Context) {
	info := struct {
		NetworkUrl string `form:"networkUrl" binding:"required"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	net, err := model.GetNetworkfromNets(
		model.NewCaUserFromDomainName(info.NetworkUrl).NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgs := []response.Organization{}
	if len(net.Organizations) >= 2 {
		orgs = response.NewOrgs(net.Organizations[1:])
	}

	response.Ok().
		SetPayload(orgs).
		Result(c.JSON)
}