package router

import (
	"mictract/api"

	"github.com/gin-gonic/gin"
)

func GetRouter() (router *gin.Engine) {
	router = gin.Default()

	NetworkRouter := router.Group("network")
	{
		NetworkRouter.POST("/", api.CreateNetwork)
		NetworkRouter.GET("/", api.ListNetworks)
		NetworkRouter.DELETE("/:id", api.DeleteNetwork)
		NetworkRouter.GET("/:id", api.GetNetwork)

		NetworkRouter.POST("/addOrg", api.AddOrg)
		NetworkRouter.POST("/addPeer", api.AddPeer)
		NetworkRouter.POST("/addOrderer", api.AddOrderer)
		NetworkRouter.POST("/addChannel", api.AddChannel)
	}

	MysqlRouter := router.Group("mysql")
	{
		MysqlRouter.POST("/", api.CreateMysql)
		MysqlRouter.DELETE("/", api.RemoveMysql)
	}

	return
}
