package response

type Network struct {
	ID 				int 			`json:"id"`
	Nickname 		string 			`json:"nickname"`
	Consensus 		string 			`json:"consensus"`
	TlsEnabled 		bool 			`json:"tlsEnabled"`
	Status 			string 			`json:"status"`
	CreateTime 		string 			`json:"createTime"`
	Orderers 		[]Orderer 		`json:"orderers"`
	Organizations 	[]Organization 	`json:"organizations"`
	Users 			[]User 			`json:"users"`
	Channels 		[]Channel		`json:"channels"`
}

