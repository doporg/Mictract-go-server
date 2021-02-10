package service

import (
	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"
	"github.com/pkg/errors"

	"mictract/model"
	"mictract/utils"
)

func Register(mspClient *msp.Client, causer model.CaUser) error {

	request := &msp.RegistrationRequest{
		Name:   causer.Username,
		Type:   causer.MspType,
		Secret: causer.Password,
	}

	_, err := mspClient.Register(request)
	if err != nil {
		return errors.WithMessage(err, "fail to register "+causer.Username)
	}
	return nil
}

// EnrollUser enroll 一个已经注册的用户并保存相关信息
// username、networkName、orgName、mspType用于生成保存信息用的路径
// isTLS 是否是用于TLS的证书？
func Enroll(mspClient *msp.Client, isTLS bool, causer model.CaUser) error {
	var err error
	if isTLS {
		err = mspClient.Enroll(causer.Username, msp.WithSecret(causer.Password), msp.WithProfile("tls"))
	} else {
		err = mspClient.Enroll(causer.Username, msp.WithSecret(causer.Password))
	}
	if err != nil {
		return errors.WithMessage(err, "fail to enroll "+causer.Username)
	}

	resp, err := mspClient.GetSigningIdentity(causer.Username)
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

	err = utils.SaveCertAndPrivkey(cainfo.CAChain, cert, privkey, isTLS, causer)
	if err != nil {
		return errors.WithMessage(err, "fail to store info")
	}

	return nil
}
