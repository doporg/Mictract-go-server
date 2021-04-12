package request

//type AddNetwork struct {
//	// orgs[0] 个Orderer, org1 有 Orgs[1] 个peer
//	Orgs         []int `json:"orgs" binding:"required"`
//	Consensus	string `json:"consensus" binding:"required"`
//}
type AddNetworkReq struct {
	Nickname    string `form:"nickname" json:"nickname" binding:"required"`
	Consensus	string `form:"consensus" json:"consensus" binding:"required"`
	OrdererCount int 	`form:"ordererCount" form:"ordererCount" binding:"required"`
	PeerCounts	[]int	`form:"peerCounts" json:"peerCounts" binding:"required"`
	OrgNicknames []string `form:"organizationNicknames" json:"organizationNicknames" binding:"required"`
	TlsEnabled	bool	`form:"tlsEnalbed"`
}

type AddOrgReq struct {
	NetworkID 	int 	`form:"networkID" json:"networkID" binding:"required"`
	PeerCount 	int 	`form:"peerCount" json:"peerCount" binding:"required"`
	Nickname   	string 	`form:"nickname" json:"nickname" binding:"required"`
}

type AddOrdererReq struct {
	NetworkID 	 		int 		`form:"networkID" json:"networkID" binding:"required"`
	OrdererCount 	 	int 		`form:"ordererCount" json:"ordererCount" binding:"required"`
}

type AddPeerReq struct {
	OrganizationID int		`form:"organizationID" json:"organizationID" binding:"required"`
	PeerCount 	   int 		`form:"peerCount" json:"peerCount" binding:"required"`
}

type AddChannelReq struct {
	Nickname        string 		`form:"nickname" json:"nickname"  binding:"required"`
	NetworkID		int		 	`form:"networkID" json:"networkID" binding:"required"`
	OrganizationIDs	[]int	 	`form:"organizationIDs" json:"organizationIDs" binding:"required"`
}

