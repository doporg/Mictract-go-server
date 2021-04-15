package response

type Channel struct {
	ChannelID 		int 		`json:"id"`
	NetworkID		int 		`json:"networkID"`
	Nickname 		string 		`json:"nickname"`
	Organizations 	[]string 	`json:"organizations"`
	Status 			string 		`json:"status"`
}
