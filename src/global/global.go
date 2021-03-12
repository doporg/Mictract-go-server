package global

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	// global variables go here.
	DB					*gorm.DB
	Logger				*zap.Logger
	SDKs				map[string]*fabsdk.FabricSDK
	Nets				map[string]interface{}
	K8sClientset		*kubernetes.Clientset
	K8sRestConfig		*rest.Config
	K8sInformer			cache.SharedIndexInformer
	K8sLister			v1.PodNamespaceLister
)

var (
	k8sInformerFactory	informers.SharedInformerFactory
	k8sInformerStopCh	chan struct{}
)

func init() {
	initLogger()
	//initDB()
	initK8s()
	initSDKs()
}

func Close() {
	//closeDB()
	//closeK8sInformer()
	closeSDKs()
}
