package api

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"mictract/enum"
	"mictract/model/response"
	"mictract/service/factory/sdk"
	"net/http"
)

// GET /api/sdk/
func GetSDKByUserID(c *gin.Context)  {
	userIDInt, ok := c.Get("userID")
	if !ok {
		response.Err(http.StatusForbidden, enum.CodeErrBadArgument).
			SetMessage("userID not found").
			Result(c.JSON)
		return
	}
	userID := userIDInt.(int)
	if userID <= 0 {
		response.Err(http.StatusForbidden, enum.CodeErrBadArgument).
			SetMessage("Wrong user id or network administrator").
			Result(c.JSON)
		return
	}

	configObj := sdk.NewSDKFactory().NewSDKConfigByUserID(userID)
	sdkconfig, err 	:= yaml.Marshal(configObj)
	if err != nil {
		response.Err(http.StatusForbidden, enum.CodeErrBadArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	response.Ok().SetPayload(sdkconfig).Result(c.JSON)
}
