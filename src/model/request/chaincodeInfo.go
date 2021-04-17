package request

type InvokeCCReq struct {
	ChaincodeID int 		`form:"id" json:"id" binding:"required"`
	PeerURLs	[]string	`form:"peerURLs" json:"peerURLs" binding:"required"`
	Args 		[]string 	`form:"args" json:"args" binding:"required"`
	// init query execute
	InvokeType	string 		`form:"invokeType" json:"invokeType" binding:"required"`
	UserID 		int 		`form:"userID" json:"userID" binding:"required"`
}