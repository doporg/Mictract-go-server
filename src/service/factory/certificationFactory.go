package factory

import (
	"mictract/dao"
	"mictract/model"
)

type CertificationFactory struct {
}

func NewCertificationFactory() *CertificationFactory {
	return &CertificationFactory{}
}

func (cf *CertificationFactory) NewCertification(cu *model.CaUser, cert, privkey string, isTLS bool) (*model.Certification, error) {
	return cf.newCertification(cu.ID,cu.NetworkID, cu.Type, cu.Nickname, cert, privkey, isTLS)
}

func (cf *CertificationFactory) NewCACertification(org *model.Organization, cert, privkey string) (*model.Certification, error) {
	return cf.newCertification(-1, org.NetworkID,  org.GetCAID(), org.GetCAID(), cert, privkey, false)
}

func (cf *CertificationFactory) newCertification(userID, networkID int, userType, nickname, cert, privkey string, isTLS bool) (*model.Certification, error) {
	ret := &model.Certification{
		UserID: 		userID,
		NetworkID: 		networkID,
		UserType: 		userType,
		Nickname: 		nickname,
		Certification: 	cert,
		PrivateKey: 	privkey,
		IsTLS: 			isTLS,
	}

	if err := dao.InsertCertification(ret); err != nil {
		return ret, err
	}
	return ret, nil
}

