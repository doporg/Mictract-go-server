package request

// PageInfo describes which page and how large page size is requested.
// Note that Page should >= 1.
type PageInfo struct {
	Page		int `json:"page"`
	PageSize	int `json:"pageSize" binding:"required"`
}