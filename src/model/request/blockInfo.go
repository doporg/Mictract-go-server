package request

type BlockInfo struct {
	ChannelID 	int 	`form:"channelID" json:"channelID" binding:"required"`
	// if blockID == -1 return blockHeight
	BlockID 	int	   	`form:"blockID" json:"blockID"`
}