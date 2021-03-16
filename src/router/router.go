package router

import (
	"github.com/gin-gonic/gin"
	"mictract/api"
)

func GetRouter() (router *gin.Engine) {
	router = gin.Default()

	APIRoute := router.Group("api")

	NetworkRouter := APIRoute.Group("network")
	{
		NetworkRouter.POST("/", api.CreateNetwork)
		NetworkRouter.GET("/", api.ListNetworks)
		NetworkRouter.DELETE("/", api.DeleteNetwork)
		NetworkRouter.GET("/:id", api.GetNetwork)
	}

	ChannelRouter := APIRoute.Group("channel")
	{
		ChannelRouter.POST("/", api.AddChannel)
		ChannelRouter.GET("/", api.GetChannelInfo)
	}

	OrganizationRouter := APIRoute.Group("organization")
	{
		OrganizationRouter.POST("/", api.AddOrg)
	}

	PeerRouter := APIRoute.Group("peer")
	{
		PeerRouter.POST("/", api.AddPeer)
	}

	OrdererRouter := APIRoute.Group("orderer")
	{
		OrdererRouter.POST("/", api.AddOrderer)
	}

	BlockRouter := APIRoute.Group("block")
	{
		BlockRouter.GET("/", api.GetBlockByBlockID)
	}

	return
}
