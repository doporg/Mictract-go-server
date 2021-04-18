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
		NetworkRouter.GET("/:id", api.GetNetworkByID)
	}

	ChannelRouter := APIRoute.Group("channel")
	{
		ChannelRouter.POST("/", api.AddChannel)
		ChannelRouter.GET("/", api.ListChannels)
		ChannelRouter.GET("/:id", api.GetChannelByID)
	}

	OrganizationRouter := APIRoute.Group("organization")
	{
		OrganizationRouter.POST("/", api.AddOrg)
		OrganizationRouter.GET("/", api.ListOrganizations)
		OrganizationRouter.GET("/:id", api.GetOrganizationByID)
	}

	UserRouter := APIRoute.Group("user")
	{
		UserRouter.POST("/", api.CreateUser)
		UserRouter.GET("/", api.ListUsers)
		UserRouter.GET("/:id", api.GetUserByID)
		UserRouter.DELETE("/", api.DeleteUser)
	}

	PeerRouter := APIRoute.Group("peer")
	{
		PeerRouter.POST("/", api.AddPeer)
		PeerRouter.GET("/", api.ListPeersByOrganization)
		PeerRouter.GET("/:id", api.GetPeerByID)

		PeerChannelRouter := PeerRouter.Group("channel")
		{
			PeerChannelRouter.POST("/", api.JoinPeerToChannel)
			PeerChannelRouter.GET("/", api.ListChannelsInPeer)
		}
	}

	OrdererRouter := APIRoute.Group("orderer")
	{
		OrdererRouter.POST("/", api.AddOrderer)
		OrdererRouter.GET("/", api.ListOrderersByNetwork)
		OrdererRouter.GET("/:id", api.GetOrdererByID)
	}

	BlockRouter := APIRoute.Group("block")
	{
		BlockRouter.GET("/", api.GetBlockByBlockID)
	}

	CCRouter := APIRoute.Group("chaincode")
	{
		CCRouter.POST("/", api.CreateChaincode)
		CCRouter.GET("/", api.ListChaincodes)

		CCRouter.POST("/install", api.InstallChaincode)
		CCRouter.POST("/approve", api.ApproveChaincode)
		CCRouter.POST("/commit", api.CommitChaincode)
		CCRouter.POST("/start", api.StartChaincodeEntity)
		// CCRouter.POST("/invoke", api.InvokeChaincode)
	}

	TxRouter := APIRoute.Group("transaction")
	{
		TxRouter.POST("/", api.InvokeChaincode)
		TxRouter.GET("/", api.ListTransaction)
		TxRouter.GET("/:id", api.GetTransactionInBlockchain)
		TxRouter.DELETE("/", api.DeleteTransaction)
	}
	return
}
