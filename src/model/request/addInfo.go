package request

type AddBasicNetwork struct {
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


