package response

import "mictract/model"

type Orderer struct {
	Name    string `json:"name"`
}

func NewOrderer(o model.CaUser) *Orderer {
	return &Orderer{o.GetName()}
}

func NewOrderers(os []model.CaUser) []Orderer {
	orderers := []Orderer{}
	for _, o := range os {
		orderers = append(orderers, *NewOrderer(o))
	}
	return orderers
}