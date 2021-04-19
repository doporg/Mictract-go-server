package response

type Organization struct {
	Nickname		string		`json:"nickname"`
	OrganizationID 	int 		`json:"id"`
	NetworkID 		int 		`json:"networkID"`
	Peers 			[]Peer 		`json:"peers"`
	Users 			[]User 		`json:"users"`
	Status 			string 		`json:"status"`
}
