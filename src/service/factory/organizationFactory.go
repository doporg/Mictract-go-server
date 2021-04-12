package factory

import (
	"github.com/pkg/errors"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
	"time"
)

type OrganizationFactory struct {
}

func NewOrganizationFactory() *OrganizationFactory {
	return &OrganizationFactory{}
}

func (orgf *OrganizationFactory)NewOrganization(netID int, nickname string) (*model.Organization, error) {
	return orgf.newOrganization(netID, false, nickname)
}

func (orgf *OrganizationFactory)NewOrdererOrganization(netID int, nickname string) (*model.Organization, error) {
	return orgf.newOrganization(netID, true, nickname)
}

func (orgf *OrganizationFactory) newOrganization(netID int, isOrdOrg bool, nickname string) (*model.Organization, error) {
	// 1. TODO: check netID exists or not
	net, _ := dao.FindNetworkByID(netID)
	if net.Status == enum.StatusError {
		return &model.Organization{}, errors.New("Failed to call NewOrganization, network status is abnormal")
	}

	// 2. new
	org := &model.Organization{
		NetworkID: 		netID,
		Nickname: 		nickname,
		Status: 		enum.StatusStarting,
		CreatedAt: 		time.Now(),
		IsOrdererOrg: 	isOrdOrg,
	}

	// 3. insert into db
	if err := dao.InsertOrganization(org); err != nil {
		return &model.Organization{}, err
	}

	return org, nil
}
