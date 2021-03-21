package response

import (
	"fmt"
	"mictract/model"
)

type User struct {
	Name 			string `json:"name"`
	Role 			string `json:"role"`
	Nickname 		string `json:"nickname"`
	Organization 	string `json:"organization"`
	Network			string `json:"network"`
}

func NewUser(u model.User) User {
	user := model.NewCaUserFromDomainName(u.Username)
	return User{
		Name: u.Username,
		Role: u.UserType,
		Nickname: u.Nickname,
		Organization: fmt.Sprintf("org%d.net%d.com", user.OrganizationID, user.NetworkID),
		Network: fmt.Sprintf("net%d.com", user.NetworkID),
	}
}

func NewUsers(us []model.User) []User {
	usersResp := []User{}
	for _, u := range us {
		usersResp = append(usersResp, NewUser(u))
	}
	return usersResp
}