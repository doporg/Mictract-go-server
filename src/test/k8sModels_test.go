package test

import (
	"mictract/global"
	"mictract/model/kubernetes"
	"testing"
)

var (
	ordererCA	= kubernetes.NewOrdererCA(1)
	org1PeerCA	= kubernetes.NewPeerCA(1, 1)

	orderer1 	= kubernetes.NewOrderer(1, 1)
	org1Peer1	= kubernetes.NewPeer(1, 1, 1)
	org1Peer2	= kubernetes.NewPeer(1, 1, 2)

	models		= []kubernetes.K8sModel{
		org1PeerCA, org1Peer1, org1Peer2,
		ordererCA, orderer1,
	}
)

func TestCreateK8sModels(t *testing.T) {
	for _, v := range models {
		v.Create(global.K8sClientset)
	}
}

func TestDeleteK8sModels(t *testing.T) {
	for _, v := range models {
		v.Delete(global.K8sClientset)
	}
}