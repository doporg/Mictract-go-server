package response

type Organization struct {
	Nickname		string		`json:"nickname"`
	OrganizationID 	int 		`json:"id"`
	NetworkID 		int 		`json:"networkID"`
	Peers 			[]string 	`json:"peers"`
	Users 			[]string 	`json:"users"`
	Status 			string 		`json:"status"`
}
