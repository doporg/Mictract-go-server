package request

type BlockInfo struct {
	NetID int `json:"netid" binding:"required"`
	// >= 0
	ChannelID int `json:"channelid" binding:"required"`
	BlockID	uint64 `json:"blockid"`
}
