package request

type ChannelInfo struct {
	NetID	int	`form:"netid" binding:"required"`
	ChannelID	int	`form:"channelid" binding:"required"`
}
