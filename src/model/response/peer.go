package response

import "mictract/model"

type Peer struct {
	Name  	string `json:"name"`
}

func NewPeer(p model.Peer) *Peer {
	return &Peer{p.Name}
}

func NewPeers(ps []model.Peer) []Peer {
	peers := []Peer{}
	for _, p := range ps {
		peers = append(peers, *NewPeer(p))
	}
	return peers
}
