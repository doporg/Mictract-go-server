package response

import "mictract/model"

type User struct {
	Name string `json:"name"`
	Role string `json:"role"`
	Nickname string `json:"nickname"`
	Organization string `json:"organization"`
}

func NewUser(u model.User) User {
	return User{
		Name: u.Username,
		Role: u.UserType,
		Nickname: u.Nickname,
		Organization: u.OrgName,
	}
}

func NewUsers(us []model.User) []User {
	usersResp := []User{}
	for _, u := range us {
		usersResp = append(usersResp, NewUser(u))
	}
	return usersResp
}