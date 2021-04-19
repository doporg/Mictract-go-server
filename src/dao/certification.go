package dao

import (
	"github.com/pkg/errors"
	"mictract/global"
	"mictract/model"
)

func InsertCertification(cert *model.Certification) error {
	return global.DB.Create(cert).Error
}

func FindCACertByOrganizationID(orgID int) (*model.Certification, error) {
	org, err := FindOrganizationByID(orgID)
	if err != nil {
		return &model.Certification{}, err
	}
	certs := []model.Certification{}
	if err := global.DB.
		Where("user_type = ?", org.GetCAID()).
		Find(&certs).Error; err != nil {
		return &model.Certification{}, err
	}
	if len(certs) < 1 {
		return &model.Certification{}, errors.New("no cert in db")
	}
	return &certs[0], nil
}

func FindCertsByUserID(userID int, isTLS bool) ([]model.Certification, error) {
	certs := []model.Certification{}
	if err := global.DB.
		Where("user_id = ? and is_tls = ?", userID, isTLS).
		Find(&certs).Error; err != nil {
		return certs, err
	}
	return certs, nil
}

func FindCertByUserID(userID int, isTLS bool) (*model.Certification, error) {
	certs := []model.Certification{}
	if err := global.DB.
		Where("user_id = ? and is_tls = ?", userID, isTLS).
		Find(&certs).Error; err != nil {
		return &model.Certification{}, err
	}
	if len(certs) < 1 {
		return &model.Certification{}, errors.New("no cert in user")
	}
	return &certs[0], nil
}

// not tls cert
func FindSystemUserCertByOrg(org *model.Organization) (*model.Certification, error) {
	adminUser, err := FindSystemUserInOrganization(org.ID)
	if err != nil {
		return &model.Certification{}, err
	}
	certs := []model.Certification{}
	if err := global.DB.
		Where("user_id = ? and is_tls = ?", adminUser.ID, false).
		Find(&certs).Error; err != nil {
		return &model.Certification{}, err
	}
	if len(certs) != 1 {
		return &model.Certification{}, errors.New("There should be only one system user certificate")
	}
	return &certs[0], nil
}

func FindAllCerts() ([]model.Certification, error){
	certs := []model.Certification{}
	if err := global.DB.Where("user_type in ? and nickname <> 'system-user'", []string{"admin", "user"}).Find(&certs).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all certs")
	}
	return certs, nil
}

func DeleteCertByID(certID int) error {
	// TODO: invoke
	return  global.DB.Where("id = ?", certID).Delete(&model.Certification{}).Error
}

func DeleteAllCertificationsInNetwork(netID int) error {
	return global.DB.Where("network_id = ?", netID).Delete(&model.Certification{}).Error
}
