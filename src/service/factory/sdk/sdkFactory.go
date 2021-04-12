package sdk

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"gopkg.in/yaml.v3"
	mConfig "mictract/config"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"os"
	"path/filepath"
)

type SDKFactory struct {
}

func NewSDKFactory() *SDKFactory {
	return &SDKFactory{}
}

func (sdkf *SDKFactory)NewOrgSDKByOrganizationID(orgID int) (*fabsdk.FabricSDK, error) {
	configObj 		:= sdkf.newSDKConfigByOrganizationID(orgID)
	org, _ 			:= dao.FindOrganizationByID(orgID)
	sdkconfig, err 	:= yaml.Marshal(configObj)
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}

	// global.Logger.Info(string(sdkconfig))
	// for debug
	go sdkf._store(filepath.Join(mConfig.LOCAL_BASE_PATH, model.GetNetworkNameByID(org.NetworkID), "sdk"),
		fmt.Sprintf("%s.yaml", org.GetName()),
		string(sdkconfig),
	)

	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}
	sdkf.updateSDK(org.GetName(), sdk)
	return sdk, nil
}

func (sdkf *SDKFactory)NewCompleteSDKByNetworkID(netID int) (*fabsdk.FabricSDK, error) {
	configObj 		:= sdkf.newSDKConfigByNetworkID(netID)
	sdkconfig, err 	:= yaml.Marshal(configObj)
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}

	// global.Logger.Info(string(sdkconfig))
	// for debug
	go sdkf._store(filepath.Join(mConfig.LOCAL_BASE_PATH, model.GetNetworkNameByID(netID), "sdk"),
		fmt.Sprintf("%s.yaml", model.GetNetworkNameByID(netID)),
		string(sdkconfig),
	)

	sdk, err := fabsdk.New(config.FromRaw(sdkconfig, "yaml"))
	if err != nil {
		return &fabsdk.FabricSDK{}, err
	}
	sdkf.updateSDK(model.GetNetworkNameByID(netID), sdk)
	return sdk, nil
}

func (sdkf *SDKFactory)newSDKConfigByNetworkID(netID int) *model.SDKConfig {
	netPeers, _			:= dao.FindAllPeersInNetwork(netID)
	orgs, _ 			:= dao.FindAllOrganizationsInNetwork(netID)
	sdkCSF 				:= NewSDKConfigSonFactory()
	sdkconfig 			:= sdkf.newCommonSDKConfigByNetworkID(netID)
	sdkconfig.Client	= sdkCSF.NewSDKConfigClient(&orgs[0])

	for _, org := range orgs {
		sdkconfig.Organizations[org.GetName()] 			= sdkCSF.NewSDKConfigOrganization(&org)
		sdkconfig.CertificateAuthorities[org.GetCAID()] = sdkCSF.NewSDKConfigCA(&org)
	}
	for _, peer := range netPeers {
		sdkconfig.Peers[peer.GetName()] = sdkCSF.NewSDKConfigNode(&peer)
	}

	return sdkconfig
}

func (sdkf *SDKFactory)newSDKConfigByOrganizationID(orgID int) *model.SDKConfig {
	sdkCSF 											:= NewSDKConfigSonFactory()
	org, _ 											:= dao.FindOrganizationByID(orgID)
	peers, _ 										:= dao.FindAllPeersInOrganization(orgID)
	sdkconfig 										:= sdkf.newCommonSDKConfigByNetworkID(org.NetworkID)
	sdkconfig.Client 								= sdkCSF.NewSDKConfigClient(org)
	sdkconfig.Organizations[org.GetName()] 			= sdkCSF.NewSDKConfigOrganization(org)
	sdkconfig.CertificateAuthorities[org.GetCAID()] = sdkCSF.NewSDKConfigCA(org)

	for _, peer := range peers {
		sdkconfig.Peers[peer.GetName()] = sdkCSF.NewSDKConfigNode(&peer)
	}

	return sdkconfig
}

// include orderers channels
func (sdkf *SDKFactory)newCommonSDKConfigByNetworkID(netID int) *model.SDKConfig {
	orderers, _ 	:= dao.FindAllOrderersInNetwork(netID)
	chs, _ 			:= dao.FindAllChannelsInNetwork(netID)
	sdkCSF 			:= NewSDKConfigSonFactory()

	sdkconfig := &model.SDKConfig{
		Version: 				"GodBlessNoBug",
		Client: 				&model.SDKConfigClient{},
		Organizations:          map[string]*model.SDKConfigOrganization{},
		Orderers:               map[string]*model.SDKConfigNode{},
		Peers:                  map[string]*model.SDKConfigNode{},
		Channels:               map[string]*model.SDKConfigChannel{},
		CertificateAuthorities: map[string]*model.SDKConfigCA{},
	}

	// orderers
	for _, orderer := range orderers {
		sdkconfig.Orderers[orderer.GetName()] = sdkCSF.NewSDKConfigNode(&orderer)
	}

	// channels
	sdkconfig.Channels["_default"] = sdkCSF.NewDefaultSDKConfigChannel(netID)
	for _, ch := range chs {
		sdkconfig.Channels[ch.GetName()] = sdkCSF.NewSDKConfigChannel(&ch)
	}

	return sdkconfig
}

func (sdkf *SDKFactory)updateSDK(key string, sdk *fabsdk.FabricSDK) {
	oldSDK, ok := global.SDKs[key]
	if ok {
		oldSDK.Close()
	}
	global.SDKs[key] = sdk
}

func (sdkf *SDKFactory)_store(path, filename, s string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
	f, _ := os.Create(filepath.Join(path, filename))
	_, _ = f.WriteString(string(s))
	_ = f.Close()
}
