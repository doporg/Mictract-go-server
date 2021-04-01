package response

import "mictract/model"

type Orderer struct {
	Name    string `json:"name"`
}

func NewOrderer(o model.Order) *Orderer {
	return &Orderer{o.Name}
}

func NewOrderers(os []model.Order) []Orderer {
	orderers := []Orderer{}
	for _, o := range os {
		orderers = append(orderers, *NewOrderer(o))
	}
	return orderers
}