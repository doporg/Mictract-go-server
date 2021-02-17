package global

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	// global variables go here.
	DB				*gorm.DB
	Logger			*zap.Logger
	SDKs			map[string]*fabsdk.FabricSDK
	K8sClientset	*kubernetes.Clientset
	K8sRestConfig	*rest.Config
)

func init() {
	initLogger()
	// initDB()
	initK8sClient()
}

func Close() {
	closeDB()
}
