package response

import "mictract/model"

type Peer struct {
	Nickname  	string 	`json:"nickname"`
	ID 			int 	`json:"id"`
}

func NewPeer(p model.CaUser) *Peer {
	return &Peer{
		Nickname: 	p.GetName(),
		ID: 	 	p.ID,
	}
}

func NewPeers(ps []model.CaUser) []Peer {
	peers := []Peer{}
	for _, p := range ps {
		peers = append(peers, *NewPeer(p))
	}
	return peers
}
