package request

type AddNetwork struct {
	// orgs[0] 个Orderer, org1 有 Orgs[1] 个peer
	Orgs         []int `json:"orgs" binding:"required"`
	Consensus	string `json:"consensus" binding:"required"`
}

type AddOrgReq struct {
	NetID	int `json:"netid" binding:"required"`
}

type AddOrdererReq struct {
	NetID	int	`json:"netid" binding:"required"`
	Num     int `json:"num" binding:"required"`
}

type AddPeerReq struct {
	NetID	int `json:"netid" binding:"required"`
	OrgID	int `json:"orgid" binding:"required"`
	Num 	int `json:"num" binding:"required"`
}

type AddChannelReq struct {
	NetID 	int 	`json:"netid" binding:"required"`
	OrgIDs	[]int	`json:"orgids" binding:"required"`
}

