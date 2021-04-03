package request

type InvokeCCReq struct {
	PeerURLs	[]string	`form:"peerURLs" json:"peerURLs" binding:"required"`
	Args 		[]string 	`form:"args" json:"args" binding:"required"`
	// init query execute
	InvokeType	string 		`form:"invokeType" json:"invokeType" binding:"required"`
}