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
