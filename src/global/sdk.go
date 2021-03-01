package global

import "github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"

func initSDKs() {
	SDKs = make(map[string]*fabsdk.FabricSDK)
}

func closeSDKs() {
	for _, sdk := range SDKs {
		sdk.Close()
	}
}