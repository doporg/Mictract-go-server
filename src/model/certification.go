package model

type Certification struct {
	ID 				int 	`json:"id"`
	UserID			int 	`json:"user_id"`
	NetworkID 		int		`json:"network_id"`
	// user admin peer orderer org.GetCAID()
	UserType 		string 	`json:"user_type"`
	Nickname 		string 	`json:"nickname"`

	Certification   string 	`json:"certification"`
	PrivateKey 		string 	`json:"private_key"`

	IsTLS			bool	`json:"is_tls"`
}