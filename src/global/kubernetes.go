package global

import (
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"mictract/config"
	"time"
)

func initK8s() {
	initK8sClient()
	initK8sInformerFactory()
}

func initK8sClient() {
	var err error
	if K8sRestConfig, err = clientcmd.BuildConfigFromFlags("", config.K8S_CONFIG); err != nil {
		Logger.Error("Get kubernetes rest config error", zap.Error(err))
	}

	if K8sClientset, err = kubernetes.NewForConfig(K8sRestConfig); err != nil {
		Logger.Error("Get kubernetes clientset error", zap.Error(err))
	}
}

func initK8sInformerFactory() {
	k8sInformerFactory = informers.NewSharedInformerFactory(K8sClientset, time.Second)
	K8sInformer = k8sInformerFactory.Core().V1().Pods().Informer()
	K8sLister = k8sInformerFactory.Core().V1().Pods().Lister().Pods(metav1.NamespaceDefault)

	go k8sInformerFactory.Start(k8sInformerStopCh)

	if !cache.WaitForCacheSync(k8sInformerStopCh, K8sInformer.HasSynced) {
		Logger.Warn("Timed out waiting for cache to sync")
	}
}

func closeK8sInformer() {
	k8sInformerStopCh <- struct{}{}
	close(k8sInformerStopCh)
}