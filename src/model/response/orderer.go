package response

import "mictract/model"

type Orderer struct {
	Nickname    string 	`json:"nickname"`
	ID 			int		`json:"id"`
}

func NewOrderer(o model.CaUser) *Orderer {
	return &Orderer{
		Nickname: 	o.GetName(),
		ID: 		o.ID,
	}
}

func NewOrderers(os []model.CaUser) []Orderer {
	orderers := []Orderer{}
	for _, o := range os {
		orderers = append(orderers, *NewOrderer(o))
	}
	return orderers
}