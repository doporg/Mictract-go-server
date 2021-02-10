package service

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"

	"mictract/model"
)

func Register(mspClient *msp.Client, caUser model.CaUser) error {

	request := &msp.RegistrationRequest{
		Name:   caUser.GetUsername(),
		Type:   caUser.Type,
		Secret: caUser.Password,
	}

	_, err := mspClient.Register(request)
	if err != nil {
		return errors.WithMessage(err, "fail to register "+caUser.GetUsername())
	}
	return nil
}

// EnrollUser enroll 一个已经注册的用户并保存相关信息
// username、networkName、orgName、mspType用于生成保存信息用的路径
// isTLS 是否是用于TLS的证书？
func Enroll(mspClient *msp.Client, isTLS bool, causer model.CaUser) error {
	var err error
	username := causer.GetUsername()

	if isTLS {
		err = mspClient.Enroll(username, msp.WithSecret(causer.Password), msp.WithProfile("tls"))
	} else {
		err = mspClient.Enroll(username, msp.WithSecret(causer.Password))
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

	err = causer.BuildDir(cainfo.CAChain, cert, privkey)
	if err != nil {
		return errors.WithMessage(err, "fail to store info")
	}

	return nil
}
