package response

import (
	"mictract/model"
)

type User struct {
	UserID  		int 	`json:"id"`
	Role 			string 	`json:"role"`
	Nickname 		string 	`json:"nickname"`
	OrganizationID 	int 	`json:"organizationID"`
	NetworkID		int		`json:"networkID"`

	PrivateKey 		string 	`json:"privateKey"`
	Certificate 	string 	`json:"certificate"`
}

func NewUser(u model.CaUser) *User {
	return &User{
		UserID: u.ID,
		Role: u.Type,
		Nickname: u.Nickname,
		OrganizationID: u.OrganizationID,
		NetworkID: u.NetworkID,
		PrivateKey: u.GetPrivateKey(),
		Certificate: u.GetCert(),
	}
}

func NewUsers(us []model.CaUser) []User {
	usersResp := []User{}
	for _, u := range us {
		if u.Nickname == "system-user" {
			continue
		}
		usersResp = append(usersResp, *NewUser(u))
	}
	return usersResp
}