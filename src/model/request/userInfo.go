package request

type CreateUserReq struct {
	Nickname 		string 		`form:"nickname" json:"nickname" binding:"required"`
	Role 			string 		`form:"role" json:"role" binding:"required"`
	OrganizationID	int		 	`form:"organizationID" json:"organizationID" binding:"required"`
	Password		string 		`form:"password" json:"password" binding:"required"`
}

type DeleteUserReq struct {
	UserID 	int `form:"id" json:"id" binding:"required"`
}