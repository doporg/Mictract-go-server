package request

type ChannelInfo struct {
	NetID	int	`json:"netid" binding:"required"`
	ChannelID	int	`json:"channelid" binding:"required"`
}
