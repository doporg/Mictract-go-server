package request

// PageInfo describes which page and how large page size is requested.
// Note that Page should >= 1.
type PageInfo struct {
	Page		int `json:"page,string"`
	PageSize	int `json:"pageSize,string" binding:"required"`
}