package request

type BlockInfo struct {
	NetworkURL  string `form:"networkUrl" binding:"required"`
	ChannelName string `form:"channelName" binding:"required"`
	// if blockID == -1 return blockHeight
	BlockID 	int	   `form:"blockid"`
}