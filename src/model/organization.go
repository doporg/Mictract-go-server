package model

type Organization struct {
	Name 	string 	`json:"name"`
	Peers 	[]Peer 	`json:"peers"`
}
