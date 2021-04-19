package api

import (
	"github.com/gin-gonic/gin"
	"mictract/dao"
	"mictract/enum"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"mictract/service/factory"
	"net/http"
	"strconv"
)

// POST /api/peer
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

// GET /api/orderer
func GetPeerByID(c *gin.Context)  {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	peer, err := dao.FindCaUserByID(id)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewPeer(peer)).
		Result(c.JSON)
}

// =========================================================================================================

// POST /api/peer/channel
/* join channel
{
	"channelID": 10,
	"peerNames": ["peer10.org5.net11.com"]
}
*/

func JoinPeerToChannel(c *gin.Context)  {
	info := struct{
		ChannelID  int 		`form:"channelID" json:"channelID" binding:"required"`
		PeerNames  []string	`form:"peerNames" json:"peerNames" binding:"required"`
	}{}

	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err := dao.FindChannelByID(info.ChannelID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orderers, err := dao.FindAllOrderersInNetwork(ch.NetworkID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for _, peerName := range info.PeerNames {
		peer := factory.NewCaUserFactory().NewCaUserFromDomainName(peerName)
		if err := service.NewCaUserService(peer).JoinChannel(
			ch.ID,
			orderers[0].GetName()); err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}

	response.Ok().Result(c.JSON)
	return
}

// GET /api/peer/channel
/* list channel
{
	"peerName": "peer10.org5.net11.com"
}
payload
{
	channelID: nickname,
	channelID: nickname,
	...
}
*/

func ListChannelsInPeer(c *gin.Context) {
	info := struct {
		PeerName  string  `form:"peerName" json:"peerName" binding:"required"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	list, err := service.
		NewCaUserService(factory.NewCaUserFactory().NewCaUserFromDomainName(info.PeerName)).
		GetJoinedChannel()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().SetPayload(list).Result(c.JSON)
}