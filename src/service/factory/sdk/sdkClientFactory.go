package sdk

import (
	channelclient "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	mspclient "github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
	"mictract/model"
)

type SDKClientFactory struct {
}

func NewSDKClientFactory() *SDKClientFactory {
	return &SDKClientFactory{}
}

func (sdkCF *SDKClientFactory) NewLedgerClient(user *model.CaUser, ch *model.Channel) (*ledger.Client, error) {
	sdk, err := NewSDKFactory().NewOrgSDKByOrganizationID(user.OrganizationID)
	if err != nil {
		return &ledger.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}

	ledgerClient, err := ledger.New(sdk.ChannelContext(
		ch.GetName(),
		fabsdk.WithUser(user.GetName()),
		fabsdk.WithOrg(model.GetOrganizationNameByIDAndBool(user.OrganizationID, user.IsInOrdererOrg()))))
	if err != nil {
		return &ledger.Client{}, err
	}
	return ledgerClient, nil
}

func (sdkCF *SDKClientFactory) NewResmgmtClient(user *model.CaUser) (*resmgmt.Client, error) {
	sdk, err := NewSDKFactory().NewOrgSDKByOrganizationID(user.OrganizationID)
	if err != nil {
		return &resmgmt.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}
	resmgmtClient, err := resmgmt.New(sdk.Context(
		fabsdk.WithUser(user.GetName()),
		fabsdk.WithOrg(model.GetOrganizationNameByIDAndBool(user.OrganizationID, user.IsInOrdererOrg()))))
	if err != nil {
		return &resmgmt.Client{}, err
	}
	return resmgmtClient, nil
}

func (sdkCF *SDKClientFactory) NewChannelClient(user *model.CaUser, ch *model.Channel) (*channelclient.Client, error) {
	sdk, err := NewSDKFactory().NewOrgSDKByOrganizationID(user.OrganizationID)
	if err != nil {
		return &channelclient.Client{}, errors.WithMessage(err, "fail to get sdk ")
	}
	ccp := sdk.ChannelContext(
		ch.GetName(),
		fabsdk.WithUser(user.GetName()),
		fabsdk.WithOrg(model.GetOrganizationNameByIDAndBool(user.OrganizationID, user.IsInOrdererOrg())))
	chClient, err := channelclient.New(ccp)
	if err != nil {
		return &channelclient.Client{}, err
	}
	return chClient, nil
}

func (sdkCF *SDKClientFactory) NewMSPClient(org *model.Organization) (*mspclient.Client, error) {
	sdk, err := NewSDKFactory().NewOrgSDKByOrganizationID(org.ID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get sdk")
	}
	return mspclient.New(sdk.Context(), mspclient.WithCAInstance(org.GetCAID()), mspclient.WithOrg(org.GetName()))
}
