package model

type causer struct {
	name string `json:"name" binding:"required"`
	pw   string `json:"pw" binding:"required"`
}
