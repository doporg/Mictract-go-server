package response

type Channel struct {
	ChannelID 		int 			`json:"id"`
	NetworkID		int 			`json:"networkID"`
	Nickname 		string 			`json:"nickname"`
	Organizations 	[]Organization 	`json:"organizations"`
	Status 			string 			`json:"status"`

	// only set when call api /api/channel/:id
	Height 			uint64			`json:"height"`
}
