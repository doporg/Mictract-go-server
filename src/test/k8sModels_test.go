package test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"mictract/global"
	"mictract/model/kubernetes"
	"reflect"
	"testing"
	"time"
)

var (
	mysql		= &kubernetes.Mysql{}
	tools 		= &kubernetes.Tools{}
	ordererCA	= kubernetes.NewOrdererCA(1)
	org1PeerCA	= kubernetes.NewPeerCA(1, 1)

	orderer1 	= kubernetes.NewOrderer(1, 1)
	org1Peer1	= kubernetes.NewPeer(1, 1, 1)
	org1Peer2	= kubernetes.NewPeer(1, 1, 2)

	models		= []kubernetes.K8sModel{
		tools,
		org1PeerCA, org1Peer1, org1Peer2,
		ordererCA, orderer1,
	}
)

func TestCreateK8sModels(t *testing.T) {
	for _, v := range models {
		v.Create()
	}
}

func TestDeleteK8sModels(t *testing.T) {
	for _, v := range models {
		v.Delete()
	}
}

func TestInformer(t *testing.T) {
	global.K8sInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			pod := o.(*v1.Pod)
			if pod.Labels["app"] != "mictract" { return }
			if pod.Labels["tier"] != "tools" { return }

			fmt.Println("[informer] tools added:", pod.Status.Phase, time.Now())
		},
		UpdateFunc: func(old, new interface{}) {
			pod := new.(*v1.Pod)
			if reflect.DeepEqual(old, new) { return }
			if pod.Labels["app"] != "mictract" { return }
			if pod.Labels["tier"] != "tools" { return }

			fmt.Println("[informer] tools update:", pod)
		},
	})

	time.Sleep(time.Second)
}

func TestWatch(t *testing.T) {
	tools.Watch()
	tools.AddPodPhaseUpdateHandler(func(_ v1.PodPhase, new v1.PodPhase) {
		switch new {
		case "Running":
			global.Logger.Info("pod is running", zap.String("phase", string(new)))
		default:
			global.Logger.Info("pod phase changed", zap.String("phase", string(new)))
		}
	})

	time.Sleep(20 * time.Second)
}

func TestExecCommand(t *testing.T) {
	tests := []struct {
		command []string
		stdout	string
		isError	bool
	} {
		{ []string{"echo", "hello", "mictract"}, "hello mictract\n", false },
		{ []string{"non-exists-command"}, "", true },
	}

	if pod, err := tools.GetPod(); err == nil && pod == nil {
		tools.Create()
	}

	for _, tc := range tests {
		stdout, _, err := tools.ExecCommand(tc.command...)

		assert.Equal(t, tc.isError, err != nil)
		if err == nil {
			assert.Equal(t, tc.stdout, stdout)
		}
	}

}

func TestCreateOrdererCA(t *testing.T) {
	ordererCA.Create()
}

func TestDeleteOrdererCA(t *testing.T) {
	ordererCA.Delete()
}

func TestCreateMysql(t *testing.T) {
	mysql.Create()
}

func TestDeleteMysql(t *testing.T) {
	mysql.Delete()
}