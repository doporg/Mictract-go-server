package service

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"

	"mictract/utils"
)

func Register(mspClient *msp.Client, username, password, msptype string) error {

	request := &msp.RegistrationRequest{
		Name:   username,
		Type:   msptype,
		Secret: password,
	}

	_, err := mspClient.Register(request)
	if err != nil {
		return errors.WithMessage(err, "fail to register "+username)
	}
	return nil
}

// EnrollUser enroll 一个已经注册的用户并保存相关信息
// username、networkName、orgName、mspType用于生成保存信息用的路径
// isTLS 是否是用于TLS的证书？
func Enroll(mspClient *msp.Client, username, password, networkName, orgName, mspType string, isTLS bool) error {
	var err error
	if isTLS {
		err = mspClient.Enroll(username, msp.WithSecret(password), msp.WithProfile("tls"))
	} else {
		err = mspClient.Enroll(username, msp.WithSecret(password))
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

	err = utils.SaveCertAndPrivkey(cainfo.CAChain, cert, privkey, username, networkName, orgName, mspType, isTLS)
	if err != nil {
		return errors.WithMessage(err, "fail to store info")
	}

	return nil
}
