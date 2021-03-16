package router

import (
	"github.com/gin-gonic/gin"
	"mictract/api"
)

func GetRouter() (router *gin.Engine) {
	router = gin.Default()

	NetworkRouter := router.Group("network")
	{
		NetworkRouter.POST("/", api.CreateNetwork)
		NetworkRouter.GET("/", api.ListNetworks)
		NetworkRouter.DELETE("/:id", api.DeleteNetwork)
		NetworkRouter.GET("/:id", api.GetNetwork)

		//NetworkRouter.POST("/addOrg", api.AddOrg)
		NetworkRouter.POST("/addPeer", api.AddPeer)
		NetworkRouter.POST("/addOrderer", api.AddOrderer)
		// NetworkRouter.POST("/addChannel", api.AddChannel)
	}

	ChannelRouter := router.Group("channel")
	{
		ChannelRouter.POST("/", api.AddChannel)
		ChannelRouter.GET("/", api.GetChannelInfo)
	}

	OrganizationRouter := router.Group("organization")
	{
		OrganizationRouter.POST("/", api.AddOrg)
	}

	PeerRouter := router.Group("peer")
	{
		PeerRouter.POST("/", api.AddPeer)
	}

	OrdererRouter := router.Group("orderer")
	{
		OrdererRouter.POST("/", api.AddOrderer)
	}

	BlockRouter := router.Group("block")
	{
		BlockRouter.GET("/", api.GetBlockByBlockID)
	}

	return
}
