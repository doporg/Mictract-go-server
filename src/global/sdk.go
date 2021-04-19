package global

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func initSDKs() {
	SDKs 		= make(map[string]*fabsdk.FabricSDK)
	AdminSigns 	= make(map[string]*msp.SigningIdentity)
}

func closeSDKs() {
	for _, sdk := range SDKs {
		sdk.Close()
	}
}