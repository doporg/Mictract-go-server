package factory

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"regexp"
	"strconv"
	"strings"
)

type CaUserFactory struct {

}

func NewCaUserFactory() *CaUserFactory {
	return &CaUserFactory{}
}

func checkStatus(orgID, netID int) error {
	net, _ := dao.FindNetworkByID(netID)
	if net.Status == enum.StatusError {
		return errors.New("Failed to call NewCaUser, network status is abnormal")
	}
	org, _ := dao.FindOrganizationByID(orgID)
	if org.Status == enum.StatusError {
		return errors.New("Failed to call NewCaUser, organization status is abnormal")
	}
	return nil
}

func (cuf *CaUserFactory)newCaUser(nickname, password, userType string, orgID, netID int, isInOrdOrg bool) (*model.CaUser, error) {
	if err := checkStatus(orgID, netID); err != nil {
		return &model.CaUser{}, err
	}

	cu := &model.CaUser{
		Type:           			userType,
		Nickname: 					nickname,
		OrganizationID: 			orgID,
		NetworkID:      			netID,
		Password:       			password,
		IsInOrdererOrganization: 	isInOrdOrg,
	}
	if err := dao.InsertCaUser(cu); err != nil {
		return &model.CaUser{}, err
	}
	return cu, nil
}

func (cuf *CaUserFactory)NewPeerCaUser(orgID, netID int, password string) (*model.CaUser, error) {
	return cuf.newCaUser("", password, "peer", orgID, netID, false)
}

func (cuf *CaUserFactory)NewOrdererCaUser(orgID, netID int, password string) (*model.CaUser, error) {
	// Note: in our rules, orderer belongs to ordererOrganization which is unique in a given network.
	return cuf.newCaUser("", password, "orderer", orgID, netID, true)
}

func (cuf *CaUserFactory)NewUserCaUser(orgID, netID int, nickname, password string, isInOrdererOrg bool) (*model.CaUser, error) {
	return cuf.newCaUser(nickname, password, "user", orgID, netID, isInOrdererOrg)
}

func (cuf *CaUserFactory)NewAdminCaUser(orgID, netID int, nickname, password string, isInOrdererOrg bool) (*model.CaUser, error) {
	return cuf.newCaUser(nickname, password, "admin", orgID, netID, isInOrdererOrg)
}

func (cuf *CaUserFactory)NewOrganizationCaUser(orgID, netID int, isInOrdererOrg bool) *model.CaUser {
	return &model.CaUser{
		OrganizationID: orgID,
		NetworkID: netID,
		IsInOrdererOrganization: isInOrdererOrg,
	}
}

// !!!NOTE: Username here means domain name.
// Example: peer1.org1.net1.com
func (cuf *CaUserFactory)NewCaUserFromDomainName(domain string) (cu *model.CaUser) {
	return cuf.NewCaUserFromDomainNameWithPassword(domain, "")
}

// Normalize username and parse it into some kind of CaUser.
func (cuf *CaUserFactory)NewCaUserFromDomainNameWithPassword(domain, password string) *model.CaUser {
	domain = strings.ToLower(domain)
	domain = strings.ReplaceAll(domain, "@", ".")
	splicedUsername := strings.Split(domain, ".")

	dotCount := strings.Count(domain, ".")
	IdExp := regexp.MustCompile("^(user|admin|peer|orderer|org|net)([0-9]+)$")
	assignIdByOrder := func(str ...*int) {
		for i, v := range str {
			if matches := IdExp.FindStringSubmatch(splicedUsername[i]); len(matches) < 2 {
				global.Logger.Error("Error occurred in matching ID", zap.String("domainName", domain))
			} else {
				*v, _ = strconv.Atoi(matches[2])
			}
		}
	}

	cu := &model.CaUser{}

	switch {
	case strings.Contains(domain, "admin"):
		cu.Type = "admin"
		if dotCount <= 2 {
			// match: admin1.net1.com
			assignIdByOrder(&cu.ID, &cu.NetworkID)
			cu.OrganizationID = -1
		} else {
			// match: admin1.org1.net1.com
			assignIdByOrder(&cu.ID, &cu.OrganizationID, &cu.NetworkID)
		}

	case strings.Contains(domain, "user"):
		cu.Type = "user"
		if dotCount <= 2 {
			// match: user1.net1.com
			assignIdByOrder(&cu.ID, &cu.NetworkID)
			cu.OrganizationID = -1
		} else {
			// match: user1.org1.net1.com
			assignIdByOrder(&cu.ID, &cu.OrganizationID, &cu.NetworkID)
		}

	case strings.Contains(domain, "peer"):
		// match: peer1.org1.net1.com
		cu.Type = "peer"
		assignIdByOrder(&cu.ID, &cu.OrganizationID, &cu.NetworkID)

	case strings.Contains(domain, "orderer"):
		// match: orderer1.net1.com
		cu.Type = "orderer"
		assignIdByOrder(&cu.ID, &cu.NetworkID)
		cu.OrganizationID = -1

	default:
		// enhance
		// match: org1.net1.com
		// match: net1.com
		if strings.Contains(domain, "org") {
			assignIdByOrder(&cu.OrganizationID, &cu.NetworkID)
		} else {
			assignIdByOrder(&cu.NetworkID)
		}
	}

	cu.Password = password
	return cu
}