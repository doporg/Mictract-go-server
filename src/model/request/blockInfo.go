package request

type BlockInfo struct {
	NetID int `form:"netid" binding:"required"`
	// >= 0
	ChannelID int `form:"channelid" binding:"required"`
	BlockID	uint64 `form:"blockid"`
}
