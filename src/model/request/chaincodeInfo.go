package request

type DelCCReq struct {
	CCID		int		`form:"ccID" json:"ccID" binding:"required"`
	ChannelName string 	`form:"channelName" json:"channelName" binding:"required"`
	NetworkUrl	string 	`form:"networkUrl" json:"networkUrl" binding:"required"`
}

type UpdateCCNickNameReq struct {
	CCID		int 	`form:"ccID" json:"ccID" binding:"required"`
	NewNickname	string 	`form:"newNickname" json:"newNickname" binding:"required"`
}

type InvokeCCReq struct {
	PeerURLs	[]string	`form:"peerURLs" json:"peerURLs" binding:"required"`
	Args 		[]string 	`form:"args" json:"args" binding:"required"`
	// init query execute
	InvokeType	string 		`form:"invokeType" json:"invokeType" binding:"required"`
}