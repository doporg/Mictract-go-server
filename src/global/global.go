package global

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// global variables go here.
	DB     *gorm.DB
	Logger *zap.Logger
	SDKs   map[string]*fabsdk.FabricSDK
)

func init() {
	initDB()
	initLogger()
}

func Close() {
	closeDB()
}
