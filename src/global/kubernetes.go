package global

import (
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"mictract/config"
)

func initK8sClient() {
	var err error
	if K8sRestConfig, err = clientcmd.BuildConfigFromFlags("", config.K8S_CONFIG); err != nil {
		Logger.Error("Get kubernetes rest config error", zap.Error(err))
	}

	if K8sClientset, err = kubernetes.NewForConfig(K8sRestConfig); err != nil {
		Logger.Error("Get kubernetes clientset error", zap.Error(err))
	}
}
