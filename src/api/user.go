package api

import (
	"github.com/gin-gonic/gin"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
	"net/http"
)

// POST /api/user

func CreateUser(c *gin.Context) {
	var info request.CreateUserReq
	var user *model.CaUser
	var org *model.Organization
	var mspClient *mspclient.Client
	var err error

	if err = c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	} else if info.Nickname == "system-user" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("can't use system-user as nickname").
			Result(c.JSON)
		return
	}

	org, err = dao.FindOrganizationByID(info.OrganizationID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	mspClient, err = sdk.NewSDKClientFactory().NewMSPClient(org)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// insert into db
	if info.Role == "user" {
		user, err = factory.NewCaUserFactory().NewUserCaUser(org.ID, org.NetworkID, info.Nickname, info.Password, org.IsOrdererOrg)
	} else if info.Role == "admin" {
		user, err = factory.NewCaUserFactory().NewAdminCaUser(org.ID, org.NetworkID, info.Nickname, info.Password, org.IsOrdererOrg)
	} else {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("only supports user and admin").
			Result(c.JSON)
		return
	}
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// regiester
	userSvc := service.NewCaUserService(user)
	if err := userSvc.Register(mspClient); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := userSvc.Enroll(mspClient, true); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	if err := userSvc.Enroll(mspClient, false); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewUser(*user)).
		Result(c.JSON)
}

// GET /api/user
// return all users
func ListUsers(c *gin.Context) {
	users, err := dao.FindAllCaUser()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
	}

	response.Ok().
		SetPayload(response.NewUsers(users)).
		Result(c.JSON)
}

// DELETE /api/user
// TODO: revoke user from ca
func DeleteUser(c *gin.Context) {
	var info request.DeleteUserReq
	var user *model.CaUser
	var org *model.Organization
	var mspClient *mspclient.Client
	var err error

	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	user, err = dao.FindCaUserByID(info.UserID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	if user.Nickname == "system-user" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("can't delete user(system-user)").
			Result(c.JSON)
		return
	}
	if user.Type == "peer" || user.Type == "orderer" {
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("only supports user and admin").
			Result(c.JSON)
		return
	}
	org, err = dao.FindOrganizationByID(user.OrganizationID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// revoke
	mspClient, err = sdk.NewSDKClientFactory().NewMSPClient(org)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	userSvc := service.NewCaUserService(user)

	if err := userSvc.Revoke(mspClient); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrCA).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := dao.DeleteCaUserByID(user.ID); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		Result(c.JSON)
}