package response

import "mictract/model"

type Peer struct {
	Name  	string `json:"name"`
}

func NewPeer(p model.CaUser) *Peer {
	return &Peer{p.GetName()}
}

func NewPeers(ps []model.CaUser) []Peer {
	peers := []Peer{}
	for _, p := range ps {
		peers = append(peers, *NewPeer(p))
	}
	return peers
}
