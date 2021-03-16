package request

// PageInfo describes which page and how large page size is requested.
// Note that Page should >= 1.
type PageInfo struct {
	Page		int `form:"page"`
	PageSize	int `form:"pageSize" binding:"required"`
}