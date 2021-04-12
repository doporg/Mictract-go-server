package service

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/global"
	"mictract/model"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
)

type CaUserService struct {
	cu *model.CaUser
}

func NewCaUserService(cu *model.CaUser) *CaUserService {
	return &CaUserService{
		cu: cu,
	}
}

func (cuSvc *CaUserService) Register(mspClient *msp.Client) error {
	// BUG!!!
	// 用户类型有orderer、peer、admin、client，没有user
	cuType := cuSvc.cu.Type
	if cuType == "user" {
		cuType = "client"
	}
	// BUG!!!

	request := &msp.RegistrationRequest{
		Name:   cuSvc.cu.GetName(),
		Type:   cuType,
		Secret: cuSvc.cu.Password,
	}

	_, err := mspClient.Register(request)
	if err != nil {
		global.Logger.Error("fial to get register ", zap.Error(err))
		// return errors.WithMessage(err, "fail to register "+cu.GetName())
	}

	return nil
}

// EnrollUser enroll 一个已经注册的用户并保存相关信息
// username、networkName、orgName、mspType用于生成保存信息用的路径
// isTLS 是否是用于TLS的证书？
func (cuSvc *CaUserService) Enroll(mspClient *msp.Client, isTLS bool) error {
	var err error
	username := cuSvc.cu.GetName()
	hosts := []string{cuSvc.cu.GetURL(), "localhost"}

	if isTLS {
		err = mspClient.Enroll(username, msp.WithSecret(cuSvc.cu.Password), msp.WithProfile("tls"), msp.WithCSR(&msp.CSRInfo{
			CN: username,
			Hosts: hosts,
		}))
	} else {
		err = mspClient.Enroll(username, msp.WithSecret(cuSvc.cu.Password), msp.WithCSR(&msp.CSRInfo{
			CN: username,
			Hosts: hosts,
		}))
	}

	if err != nil {
		return errors.WithMessage(err, "fail to enroll "+username)
	}

	resp, err := mspClient.GetSigningIdentity(username)
	if err != nil {
		return errors.WithMessage(err, "fail to get identity")
	}

	cert := resp.EnrollmentCertificate()
	privkey, err := resp.PrivateKey().Bytes()
	if err != nil {
		return errors.WithMessage(err, "fail to get private key")
	}

	cainfo, err := mspClient.GetCAInfo()
	if err != nil {
		return errors.WithMessage(err, "fail to get cacert")
	}

	// insert into db
	factory.NewCertificationFactory().NewCertification(cuSvc.cu, string(cert), string(privkey), isTLS)

	// generate msp
	if cuSvc.cu.Type == "peer" || cuSvc.cu.Type == "orderer" {
		err = cuSvc.cu.BuildDir(cainfo.CAChain, cert, privkey, isTLS)
		if err != nil {
			return errors.WithMessage(err, "fail to store info")
		}
	}

	return nil
}

func (cuSvc *CaUserService) Revoke(mspClient *msp.Client) error {
	req := &msp.RevocationRequest{
		Name: cuSvc.cu.GetName(),
		Reason: "Marx bless, no bugs",
	}

	_, err := mspClient.Revoke(req)
	if err != nil {
		return errors.WithMessage(err, "fail to revoke " + cuSvc.cu.GetName())
	}
	return nil
}

func (cuSvc *CaUserService)JoinChannel(chID int, ordererURL string) error {
	if cuSvc.cu.Type != "peer" {
		return errors.New("only support peer")
	}

	adminUser, err := dao.FindSystemUserInOrganization(cuSvc.cu.OrganizationID)
	if err != nil {
		return err
	}

	rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}

	return rc.JoinChannel(
		model.GetChannelNameByID(chID),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(ordererURL),
		resmgmt.WithTargetEndpoints(cuSvc.cu.GetName()))
}

func (cuSvc *CaUserService)GetJoinedChannel() ([]string, error) {
	if cuSvc.cu.Type != "peer" {
		return []string{}, errors.New("only support peer")
	}
	adminUser, err := dao.FindSystemUserInOrganization(cuSvc.cu.OrganizationID)
	if err != nil {
		return []string{}, err
	}
	rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		return []string{}, errors.WithMessage(err, "fail to get rc ")
	}

	resps, err := rc.QueryChannels(resmgmt.WithTargetEndpoints(cuSvc.cu.GetName()), resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return []string{}, errors.WithMessage(err, "failed to query channel for peer")
	}

	ret := []string{}
	for _, resp := range resps.Channels {
		ret = append(ret, resp.ChannelId)
	}
	return ret, nil
}

func (cuSvc *CaUserService)QueryInstalled(orgResMgmt *resmgmt.Client) ([]resmgmt.LifecycleInstalledCC, error) {
	if cuSvc.cu.Type != "peer" {
		return []resmgmt.LifecycleInstalledCC{}, errors.New("only support peer")
	}
	resps, err := orgResMgmt.LifecycleQueryInstalledCC(
		resmgmt.WithTargetEndpoints(cuSvc.cu.GetName()),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return []resmgmt.LifecycleInstalledCC{}, errors.WithMessage(err, "fail to query ")
	}

	for _, resp := range resps {
		fmt.Println(resp)
	}

	return resps, nil
}