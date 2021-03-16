package request

type CreateUserReq struct {
	Nickname string `form:"nickname" json:"nickname" binding:"required"`
	Role 	string `form:"role" json:"role" binding:"required"`
	Organization	string `form:"organization" json:"organization" binding:"required"`
	Network 	string `form:"network" json:"network" binding:"required"`
	Password	string `form:"password" json:"password" binding:"required"`
}

type DeleteUserReq struct {
	Username string `form:"url" json:"url" binding:"required"`
}