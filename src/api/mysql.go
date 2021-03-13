package api

import (
	"github.com/gin-gonic/gin"
	"mictract/model/kubernetes"
	"mictract/model/response"
)

var mysql = &kubernetes.Mysql{}
func CreateMysql(c *gin.Context) {
	// TODO: restart
	mysql.Create()

	response.Ok().
		Result(c.JSON)
}

func RemoveMysql(c *gin.Context) {
	mysql.Delete()

	response.Ok().
		Result(c.JSON)
}