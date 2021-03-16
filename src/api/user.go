package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
)

// POST /api/user

func CreateUser(c *gin.Context) {
	var info request.CreateUserReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.Role != "user" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("role temporarily only supports user").
			Result(c.JSON)
		return
	}

	netID := model.NewCaUserFromDomainName(info.Network).NetworkID
	orgUser := model.NewCaUserFromDomainName(info.Organization)
	if netID != orgUser.NetworkID {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The organization is not part of the network").
			Result(c.JSON)
		return
	}

	org, err := model.GetOrgFromNets(orgUser.OrganizationID, orgUser.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var newUserID int

	for i := len(org.Users) - 1; i >= 1; i-- {
		tmpUser := model.NewCaUserFromDomainName(org.Users[i])
		if tmpUser.Type == "user" {
			newUserID = tmpUser.UserID + 1
			break
		}
	}

	newUser := model.CaUser{
		Type: "user",
		UserID: newUserID,
		OrganizationID: org.ID,
		NetworkID: org.NetworkID,
		Password: info.Password,
	}

	// regiester
	if err := model.UpdateSDK(org.NetworkID); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	sdk, err := model.GetSDKByNetWorkID(org.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	caURL := fmt.Sprintf("ca.org%d.net%d.com", org.ID, org.NetworkID)
	orgName := fmt.Sprintf("org%d", org.ID)
	mspClient, err := mspclient.New(
		sdk.Context(),
		mspclient.WithCAInstance(caURL),
		mspclient.WithOrg(orgName))
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := newUser.Register(mspClient); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// insert into db
	if err := model.AddUser(info.Nickname, newUser.GetUsername()); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := newUser.Enroll(mspClient, true); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	if err := newUser.Enroll(mspClient, false); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	user, err := model.QueryUserByUserName(newUser.GetUsername())
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(user).
		Result(c.JSON)
}

// GET /api/user
func ListUsers(c *gin.Context) {
	users, err := model.QueryAllUser()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
	}

	response.Ok().
		SetPayload(users).
		Result(c.JSON)
}

// DELETE /api/user
// TODO: revoke user from ca
func DeleteUser(c *gin.Context) {
	var info request.DeleteUserReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	causer := model.NewCaUserFromDomainName(info.Username)

	// revoke
	if err := model.UpdateSDK(causer.NetworkID); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	sdk, err := model.GetSDKByNetWorkID(causer.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	caURL := fmt.Sprintf("ca.org%d.net%d.com", causer.OrganizationID, causer.NetworkID)
	orgName := fmt.Sprintf("org%d", causer.OrganizationID)
	mspClient, err := mspclient.New(
		sdk.Context(),
		mspclient.WithCAInstance(caURL),
		mspclient.WithOrg(orgName))
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := causer.Revoke(mspClient); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	org, err := model.GetOrgFromNets(causer.OrganizationID, causer.NetworkID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if info.Username == org.Users[0] || info.Username == org.Users[1] {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("The user is reserved by the system and cannot be deleted!").
			Result(c.JSON)
		return
	}

	isExist := false
	for _, user := range org.Users {
		if user == info.Username {
			isExist = true
			continue
		}
	}
	if !isExist {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("user not found").
			Result(c.JSON)
		return
	}

	if err := model.DelUser(causer.GetUsername()); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		Result(c.JSON)
}