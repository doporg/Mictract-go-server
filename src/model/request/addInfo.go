package request

//type AddNetwork struct {
//	// orgs[0] 个Orderer, org1 有 Orgs[1] 个peer
//	Orgs         []int `json:"orgs" binding:"required"`
//	Consensus	string `json:"consensus" binding:"required"`
//}
type AddNetworkReq struct {
	Consensus	string `form:"consensus" json:"consensus" binding:"required"`
	OrdererCount int 	`form:"ordererCount" form:"ordererCount" binding:"required"`
	PeerCounts	[]int	`form:"peerCounts" json:"peerCounts" binding:"required"`
	TlsEnabled	bool	`form:"tlsEnalbed"`
}

type AddOrgReq struct {
	NetworkUrl string `form:"networkUrl" json:"networkUrl" binding:"required"`
	PeerCount int `form:"peerCount" json:"peerCount" binding:"required"`
}

type AddOrdererReq struct {
	NetID	int	`form:"netid" binding:"required"`
	Num     int `form:"num" binding:"required"`
}

type AddPeerReq struct {
	NetID	int `form:"netid" binding:"required"`
	OrgID	int `form:"orgid" binding:"required"`
	Num 	int `form:"num" binding:"required"`
}

//type AddChannelReq struct {
//	NetID 	int 	`form:"netid" binding:"required"`
//	OrgIDs	[]int	`form:"orgids" binding:"required"`
//}

type AddChannelReq struct {
	NetworkName		string `form:"network" json:"network" binding:"required"`
	Organizations	[]string `form:"organizations" json:"organizations" binding:"required"`
}

