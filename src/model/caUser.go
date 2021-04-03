package model

import (
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"io/ioutil"
	"mictract/config"
	"mictract/enum"
	"mictract/global"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/msp"

	"github.com/pkg/errors"
)

type CaUser struct {
	ID         		int
	Nickname 		string
	OrganizationID 	int
	NetworkID      	int
	Type           	string
	Password       	string
	IsInOrdererOrganization  bool
}

func (cu *CaUser) insert() error {
	return global.DB.Create(cu).Error
}

func checkStatus(orgID, netID int) error {
	net, _ := FindNetworkByID(netID)
	if net.Status == enum.StatusError {
		return errors.New("Failed to call NewCaUser, network status is abnormal")
	}
	org, _ := FindOrganizationByID(orgID)
	if org.Status == enum.StatusError {
		return errors.New("Failed to call NewCaUser, organization status is abnormal")
	}
	return nil
}

func NewPeerCaUser(orgID, netID int, password string) (*CaUser, error) {
	if err := checkStatus(orgID, netID); err != nil {
		return &CaUser{}, err
	}

	cu := &CaUser{
		Type:           			"peer",
		OrganizationID: 			orgID,
		NetworkID:      			netID,
		Password:       			password,
		IsInOrdererOrganization: 	false,
	}
	if err := cu.insert(); err != nil {
		return &CaUser{}, err
	}
	return cu, nil
}

func NewOrdererCaUser(orgID, netID int, password string) (*CaUser, error) {
	// Note: in our rules, orderer belongs to ordererOrganization which is unique in a given network.
	// So the OrganizationID here should be defined as a negative number.
	if err := checkStatus(orgID, netID); err != nil {
		return &CaUser{}, err
	}

	cu := &CaUser{
		Type:           			"orderer",
		OrganizationID: 			orgID,
		NetworkID:      			netID,
		Password:       			password,
		IsInOrdererOrganization: 	true,
	}
	if err := cu.insert(); err != nil {
		return &CaUser{}, err
	}
	return cu, nil
}

func NewUserCaUser(orgID, netID int, nickname, password string, isInOrdererOrg bool) (*CaUser, error) {
	if err := checkStatus(orgID, netID); err != nil {
		return &CaUser{}, err
	}

	cu := &CaUser{
		Type:           			"user",
		OrganizationID: 			orgID,
		NetworkID:      			netID,
		Password:       			password,
		Nickname:					nickname,
		IsInOrdererOrganization: 	isInOrdererOrg,
	}
	if err := cu.insert(); err != nil {
		return &CaUser{}, err
	}
	return cu, nil
}

func NewAdminCaUser(orgID, netID int, nickname, password string, isInOrdererOrg bool) (*CaUser, error) {
	if err := checkStatus(orgID, netID); err != nil {
		return &CaUser{}, err
	}

	cu := &CaUser{
		Type:           			"admin",
		OrganizationID: 			orgID,
		NetworkID:      			netID,
		Password:       			password,
		Nickname: 					nickname,
		IsInOrdererOrganization: 	isInOrdererOrg,
	}
	if err := cu.insert(); err != nil {
		return &CaUser{}, err
	}
	return cu, nil
}

func NewOrganizationCaUser(orgID, netID int, isInOrdererOrg bool) *CaUser {
	return &CaUser{
		OrganizationID: orgID,
		NetworkID: netID,
		IsInOrdererOrganization: isInOrdererOrg,
	}
}

// !!!NOTE: Username here means domain name.
// Example: peer1.org1.net1.com
func NewCaUserFromDomainName(domain string) (cu *CaUser) {
	return NewCaUserFromDomainNameWithPassword(domain, "")
}

// Normalize username and parse it into some kind of CaUser.
func NewCaUserFromDomainNameWithPassword(domain, password string) *CaUser {
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

	cu := &CaUser{}

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

func (cu *CaUser) IsInOrdererOrg() bool {
	return cu.IsInOrdererOrganization
}

// jus for peer and orderer
func (cu *CaUser) GetURL() string {
	url := ""
	switch cu.Type {
	case "user", "admin":
		url = cu.GetName()
	case "peer":
		url = fmt.Sprintf("peer%d-org%d-net%d", cu.ID, cu.OrganizationID, cu.NetworkID)
	case "orderer":
		url = fmt.Sprintf("orderer%d-net%d", cu.ID, cu.NetworkID)
	}
	return url
}

func (cu *CaUser) GetName() (username string) {
	switch cu.Type {
	case "user":
		if cu.IsInOrdererOrg() {
			username = fmt.Sprintf("User%d@net%d.com", cu.ID, cu.NetworkID)
		} else {
			username = fmt.Sprintf("User%d@org%d.net%d.com", cu.ID, cu.OrganizationID, cu.NetworkID)
		}
	case "admin":
		if cu.IsInOrdererOrg() {
			username = fmt.Sprintf("Admin%d@net%d.com", cu.ID, cu.NetworkID)
		} else {
			username = fmt.Sprintf("Admin%d@org%d.net%d.com", cu.ID, cu.OrganizationID, cu.NetworkID)
		}
	case "peer":
		username = fmt.Sprintf("peer%d.org%d.net%d.com", cu.ID, cu.OrganizationID, cu.NetworkID)
	case "orderer":
		username = fmt.Sprintf("orderer%d.net%d.com", cu.ID, cu.NetworkID)
	}
	return
}

func (cu *CaUser) GetBasePath() string {
	username := cu.GetName()
	netName := fmt.Sprintf("net%d", cu.NetworkID)

	basePath := filepath.Join(config.LOCAL_BASE_PATH, netName)
	if cu.IsInOrdererOrg() {
		domainName := fmt.Sprintf("net%d.com", cu.NetworkID)
		if cu.Type == "orderer" {
			basePath = filepath.Join(basePath,
				"ordererOrganizations", domainName,
				"orderers", username,
			)
		} else {
			basePath = filepath.Join(basePath,
				"ordererOrganizations", domainName,
				"users", username,
			)
		}
	} else {
		domainName := fmt.Sprintf("org%d.net%d.com", cu.OrganizationID, cu.NetworkID)
		if cu.Type == "peer" {
			basePath = filepath.Join(basePath,
				"peerOrganizations", domainName,
				"peers", username,
			)
		} else {
			basePath = filepath.Join(basePath,
				"peerOrganizations", domainName,
				"users", username,
			)
		}
	}

	return basePath
}

func (cu *CaUser) BuildDir(cacert, cert, privkey []byte, isTLS bool) error {
	if isTLS {
		// 此段代码生成的prefixPath目录下应该只需包括msp和tls两个文件夹
		// Build TLS directory by the given CaUser.
		prefixPath := filepath.Join(cu.GetBasePath(), "tls")
		err := os.MkdirAll(prefixPath, os.ModePerm)
		if err != nil {
			return errors.WithMessage(err, prefixPath+"创建错误")
		}

		fuckName := ""
		if cu.Type == "peer" || cu.Type == "orderer" {
			fuckName = "server"
		} else {
			fuckName = "client"
		}

		// 写入三个文件 server.crt server.key ca.crt 或者 client.crt client.key ca.crt
		for _, filename := range []string{filepath.Join(prefixPath, fuckName+".crt"),
			filepath.Join(prefixPath, fuckName+".key"),
			filepath.Join(prefixPath, "ca.crt")} {
			f, err := os.Create(filename)
			if err != nil {
				return err
			}
			defer f.Close()

			if strings.HasSuffix(filename, "key") {
				_, _ = f.Write(privkey)
			} else if strings.HasSuffix(filename, "ca.crt") {
				_, _ = f.Write(cacert)
			} else {
				_, _ = f.Write(cert)
			}
		}
	} else {

		// Build MSP directory by the given CaUser.
		prefixPath := filepath.Join(cu.GetBasePath(), "msp")
		err := os.MkdirAll(prefixPath, os.ModePerm)
		if err != nil {
			return err
		}
		/*
			msp 下有四个文件夹 cacerts tlscacerts keystore signcerts
			tlscacerts 和 cacerts文件夹中的文件夹一样，我们规定一个组织只
			用一个ca
		*/
		for _, dir := range []string{
			filepath.Join(prefixPath, "cacerts"),
			filepath.Join(prefixPath, "tlscacerts"),
			filepath.Join(prefixPath, "keystore"),
			filepath.Join(prefixPath, "signcerts"),
		} {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}
		}

		orgUrl := fmt.Sprintf("org%d.net%d.com", cu.OrganizationID, cu.NetworkID)
		if cu.IsInOrdererOrg() {
			orgUrl = fmt.Sprintf("net%d.com", cu.NetworkID)
		}
		certNameSuffix := orgUrl + "-cert.pem"

		f1, err := os.Create(filepath.Join(prefixPath, "cacerts", "ca."+certNameSuffix))
		if err != nil {
			return err
		}
		defer f1.Close()
		_, _ = f1.Write(cacert)

		f2, err := os.Create(filepath.Join(prefixPath, "tlscacerts", "tlsca."+certNameSuffix))
		if err != nil {
			return err
		}
		defer f2.Close()
		_, _ = f2.Write(cacert)

		f3, err := os.Create(filepath.Join(prefixPath, "signcerts", cu.GetName()+"-cert.pem"))
		if err != nil {
			return err
		}
		defer f3.Close()
		_, _ = f3.Write(cert)

		f4, err := os.Create(filepath.Join(prefixPath, "keystore", "priv_sk"))
		if err != nil {
			return err
		}
		defer f4.Close()
		_, _ = f4.Write(privkey)

		f5, err := os.Create(filepath.Join(prefixPath, "config.yaml"))
		if err != nil {
			return err
		}
			ouconfig := `NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/<filename>
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/<filename>
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/<filename>
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/<filename>
    OrganizationalUnitIdentifier: orderer`
		_, _ = f5.Write([]byte(strings.Replace(ouconfig, "<filename>", "ca."+certNameSuffix, -1)))
	}
	return nil
}

func (cu *CaUser) GetCACert() string {
	org, err := FindOrganizationByID(cu.OrganizationID)
	if err != nil {
		global.Logger.Error("", zap.Error(err))
	}
	return org.GetCACert()
}

func (cu *CaUser) GetCert() string {
	content, err := ioutil.ReadFile(filepath.Join(cu.GetBasePath(), "msp", "signcerts", cu.GetName() + "-cert.pem"))
	if err != nil {
		global.Logger.Error("fail to read cert.pem", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) GetTLSCert(isServerTLSCert bool) string {
	filename := "client.crt"
	if isServerTLSCert {
		filename = "server.crt"
	}
	content, err := ioutil.ReadFile(filepath.Join(cu.GetBasePath(), "tls", filename))
	if err != nil {
		global.Logger.Error("fail to read cert.pem", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) GetPrivateKey() string {
	content, err := ioutil.ReadFile(filepath.Join(cu.GetBasePath(), "msp", "keystore", "priv_sk"))
	if err != nil {
		global.Logger.Error("fail to read priv_sk", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) Register(mspClient *msp.Client) error {
	// BUG!!!
	// 用户类型有orderer、peer、admin、client，没有user
	cuType := cu.Type
	if cuType == "user" {
		cuType = "client"
	}
	// BUG!!!

	request := &msp.RegistrationRequest{
		Name:   cu.GetName(),
		Type:   cuType,
		Secret: cu.Password,
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
func (cu *CaUser) Enroll(mspClient *msp.Client, isTLS bool) error {
	var err error
	username := cu.GetName()
	hosts := []string{cu.GetURL(), "localhost"}

	if isTLS {
		err = mspClient.Enroll(username, msp.WithSecret(cu.Password), msp.WithProfile("tls"), msp.WithCSR(&msp.CSRInfo{
			CN: username,
			Hosts: hosts,
		}))
	} else {
		err = mspClient.Enroll(username, msp.WithSecret(cu.Password), msp.WithCSR(&msp.CSRInfo{
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

	err = cu.BuildDir(cainfo.CAChain, cert, privkey, isTLS)
	if err != nil {
		return errors.WithMessage(err, "fail to store info")
	}

	return nil
}

func (cu *CaUser) Revoke(mspClient *msp.Client) error {

	req := &msp.RevocationRequest{
		Name: cu.GetName(),
		Reason: "Marx bless, no bugs",
	}

	_, err := mspClient.Revoke(req)
	if err != nil {
		return errors.WithMessage(err, "fail to revoke " + cu.GetName())
	}
	return nil
}


func (peer *CaUser)JoinChannel(chID int, ordererURL string) error {
	if peer.Type != "peer" {
		return errors.New("only support peer")
	}

	ch, err := FindChannelByID(chID)
	if err != nil {
		return err
	}
	org, err := FindOrganizationByID(peer.OrganizationID)
	adminUser, err := org.GetSystemUser()
	if err != nil {
		return err
	}

	rc, err := ch.NewResmgmtClient(adminUser.GetName(), org.GetName())
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}

	return rc.JoinChannel(
		ch.GetName(),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(ordererURL),
		resmgmt.WithTargetEndpoints(peer.GetName()))
}

func (peer *CaUser)GetJoinedChannel() ([]string, error) {
	if peer.Type != "peer" {
		return []string{}, errors.New("only support peer")
	}
	sdk, err:= GetSDKByNetworkID(peer.NetworkID)
	if err != nil {
		return []string{}, errors.WithMessage(err, "fail to get sdk ")
	}

	username := fmt.Sprintf("Admin1@org%d.net%d.com", peer.OrganizationID, peer.NetworkID)
	global.Logger.Info(fmt.Sprintf("Obtaining %s's user certificate", username))
	rcp := sdk.Context(fabsdk.WithUser(username), fabsdk.WithOrg(fmt.Sprintf("org%d", peer.OrganizationID)))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		return []string{}, errors.WithMessage(err, "fail to get rc ")
	}

	resps, err := rc.QueryChannels(resmgmt.WithTargetEndpoints(peer.GetName()), resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return []string{}, errors.WithMessage(err, "failed to query channel for peer")
	}

	ret := []string{}
	for _, resp := range resps.Channels {
		ret = append(ret, resp.ChannelId)
	}
	return ret, nil
}

func (peer *CaUser)QueryInstalled(orgResMgmt *resmgmt.Client) ([]resmgmt.LifecycleInstalledCC, error) {
	if peer.Type != "peer" {
		return []resmgmt.LifecycleInstalledCC{}, errors.New("only support peer")
	}
	resps, err := orgResMgmt.LifecycleQueryInstalledCC(
		resmgmt.WithTargetEndpoints(peer.GetName()),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return []resmgmt.LifecycleInstalledCC{}, errors.WithMessage(err, "fail to query ")
	}

	for _, resp := range resps {
		fmt.Println(resp)
	}

	return resps, nil
}


func FindCaUserInOrganization(orgID, netID int, cuType string) ([]CaUser, error){
	var cus []CaUser
	if err := global.DB.
		Where("network_id = ? and organization_id = ? and type = ?", netID, orgID, cuType).Find(&cus).
		Error; err != nil {
			return []CaUser{}, err
	}
	return cus, nil
}

// user and admin
func FindCaUserInNetwork(netID int) ([]CaUser, error) {
	var cus []CaUser
	if err := global.DB.
		Where("network_id = ? and type in ?", netID, []string{"user", "admin"}).
		Find(&cus).Error; err != nil {
		return []CaUser{}, err
	}
	return cus, nil
}

// user and admin
func FindAllCaUser() ([]CaUser, error) {
	var cus []CaUser
	var err error
	if err = global.DB.Where("type in ?", []string{"user", "admin"}).Find(&cus).Error; err != nil {
		return []CaUser{}, err
	}
	return cus, nil
}

func FindCaUserByID(id int) (*CaUser, error) {
	var cus []CaUser
	if err := global.DB.Where("id = ?", id).Find(&cus).Error; err != nil {
		return &CaUser{}, err
	}
	return &cus[0], nil
}

func DeleteCaUserByID(caUserID int) error {
	return  global.DB.Where("id = ?", caUserID).Delete(&CaUser{}).Error
}