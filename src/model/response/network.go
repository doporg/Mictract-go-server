package response

type Network struct {
	NetworkID 		int 			`json:"networkID"`
	Nickname 		string 			`json:"nickname"`
	Consensus 		string 			`json:"consensus"`
	TlsEnabled 		bool 			`json:"tlsEnabled"`
	Status 			string 			`json:"status"`
	CreateTime 		string 			`json:"createTime"`
	Orderers 		[]string 		`json:"orderers"`
	Organizations 	[]Organization 	`json:"organizations"`
	Users 			[]User 			`json:"users"`
	Channels 		[]Channel		`json:"channels"`
}

