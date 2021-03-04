package model

import (
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"mictract/config"
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
	UserID         int
	OrganizationID int
	NetworkID      int
	Type           string
	Password       string
}

func NewPeerCaUser(peerID, orgID, netID int, password string) *CaUser {
	return &CaUser{
		Type:           "peer",
		UserID:         peerID,
		OrganizationID: orgID,
		NetworkID:      netID,
		Password:       password,
	}
}

func NewOrdererCaUser(ordererID, netID int, password string) *CaUser {
	// Note: in our rules, orderer belongs to ordererOrganization which is unique in a given network.
	// So the OrganizationID here should be defined as a negative number.
	return &CaUser{
		Type:           "orderer",
		UserID:         ordererID,
		OrganizationID: -1,
		NetworkID:      netID,
		Password:       password,
	}
}

func NewUserCaUser(userID, orgID, netID int, password string) *CaUser {
	return &CaUser{
		Type:           "user",
		UserID:         userID,
		OrganizationID: orgID,
		NetworkID:      netID,
		Password:       password,
	}
}

func NewAdminCaUser(userID, orgID, netID int, password string) *CaUser {
	return &CaUser{
		Type:           "admin",
		UserID:         userID,
		OrganizationID: orgID,
		NetworkID:      netID,
		Password:       password,
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
			assignIdByOrder(&cu.UserID, &cu.NetworkID)
			cu.OrganizationID = -1
		} else {
			// match: admin1.org1.net1.com
			assignIdByOrder(&cu.UserID, &cu.OrganizationID, &cu.NetworkID)
		}

	case strings.Contains(domain, "user"):
		cu.Type = "user"
		if dotCount <= 2 {
			// match: user1.net1.com
			assignIdByOrder(&cu.UserID, &cu.NetworkID)
			cu.OrganizationID = -1
		} else {
			// match: user1.org1.net1.com
			assignIdByOrder(&cu.UserID, &cu.OrganizationID, &cu.NetworkID)
		}

	case strings.Contains(domain, "peer"):
		// match: peer1.org1.net1.com
		cu.Type = "peer"
		assignIdByOrder(&cu.UserID, &cu.OrganizationID, &cu.NetworkID)

	case strings.Contains(domain, "orderer"):
		// match: orderer1.net1.com
		cu.Type = "orderer"
		assignIdByOrder(&cu.UserID, &cu.NetworkID)
		cu.OrganizationID = -1

	}

	cu.Password = password
	return cu
}

func (cu *CaUser) IsInOrdererOrg() bool {
	return cu.OrganizationID < 0
}

func (cu *CaUser) GetUsername() (username string) {
	switch cu.Type {
	case "user":
		if cu.IsInOrdererOrg() {
			username = fmt.Sprintf("User%d@net%d.com", cu.UserID, cu.NetworkID)
		} else {
			username = fmt.Sprintf("User%d@org%d.net%d.com", cu.UserID, cu.OrganizationID, cu.NetworkID)
		}
	case "admin":
		if cu.IsInOrdererOrg() {
			username = fmt.Sprintf("Admin%d@net%d.com", cu.UserID, cu.NetworkID)
		} else {
			username = fmt.Sprintf("Admin%d@org%d.net%d.com", cu.UserID, cu.OrganizationID, cu.NetworkID)
		}
	case "peer":
		username = fmt.Sprintf("peer%d.org%d.net%d.com", cu.UserID, cu.OrganizationID, cu.NetworkID)
	case "orderer":
		username = fmt.Sprintf("orderer%d.net%d.com", cu.UserID, cu.NetworkID)
	}
	return
}

func (cu *CaUser) GetBasePath() string {
	username := cu.GetUsername()
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

func (cu *CaUser) BuildDir(cacert, cert, privkey []byte) error {
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

	// Build MSP directory by the given CaUser.
	prefixPath = filepath.Join(cu.GetBasePath(), "msp")
	err = os.MkdirAll(prefixPath, os.ModePerm)
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

	f3, err := os.Create(filepath.Join(prefixPath, "signcerts", cu.GetUsername()+"-cert.com"))
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

	return nil
}

// copy file
func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}

	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func (cu *CaUser) GetOrgMspDir() string {
	netName := fmt.Sprintf("net%d", cu.NetworkID)

	basePath := filepath.Join(config.LOCAL_BASE_PATH, netName)

	if cu.OrganizationID == -1 {
		// ordererOrganizations
		basePath = filepath.Join(basePath, "ordererOrganizations", netName + ".com")
	} else {
		// peerOrganizations
		basePath = filepath.Join(basePath, "peerOrganizations", fmt.Sprintf("org%d.net%d.com", cu.OrganizationID, cu.NetworkID))
	}

	// Build MSP directory by the given CaUser.
	return filepath.Join(basePath, "msp")
}

func (cu *CaUser) GenerateOrgMsp() error {
	basePath := cu.GetOrgMspDir()


	// cacerts/ca-cert.pem
	if err := os.MkdirAll(filepath.Join(basePath, "cacerts"), os.ModePerm); err != nil {
		return err
	}
	if _, err := copy(filepath.Join(basePath, "..", "ca", "ca-cert.pem"), filepath.Join(basePath, "cacerts", "ca-cert.pem")); err != nil {
		return err
	}

	// tlscacerts/tlsca-cert.pem
	if err := os.MkdirAll(filepath.Join(basePath, "tlscacerts"), os.ModePerm); err != nil {
		return err
	}
	if _, err := copy(filepath.Join(basePath, "..", "ca", "ca-cert.pem"), filepath.Join(basePath, "tlscacerts", "tlsca-cert.pem")); err != nil {
		return err
	}

	// config.yaml
	f3, err := os.Create(filepath.Join(basePath, "config.yaml"))
	if err != nil {
		return err
	}
	ouconfig := `NodeOUs:
  Enable: true
  ClientOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: client
  PeerOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: peer
  AdminOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: admin
  OrdererOUIdentifier:
    Certificate: cacerts/ca-cert.pem
    OrganizationalUnitIdentifier: orderer`
	_, _ = f3.Write([]byte(ouconfig))

	return nil
}

func (cu *CaUser) GetCACert() string {
	content, err := ioutil.ReadFile(filepath.Join(cu.GetOrgMspDir(), "cacerts", "ca-cert.pem"))
	if err != nil {
		global.Logger.Error("fail to read ca-cert.pem", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) GetCert() string {
	content, err := ioutil.ReadFile(filepath.Join(cu.GetBasePath(), "msp", "signcerts", "cert.pem"))
	if err != nil {
		global.Logger.Error("fail to read cert.pem", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) GetPrivateKey() string {
	content, err := ioutil.ReadFile(filepath.Join(cu.GetBasePath(), "msp", "keystore", "priv_sk"))
	if err != nil {
		global.Logger.Error("fail to read cert.pem", zap.Error(err))
	}
	return string(content)
}

func (cu *CaUser) Register(mspClient *msp.Client) error {
	request := &msp.RegistrationRequest{
		Name:   cu.GetUsername(),
		Type:   cu.Type,
		Secret: cu.Password,
	}

	_, err := mspClient.Register(request)
	if err != nil {
		return errors.WithMessage(err, "fail to register "+cu.GetUsername())
	}
	return nil
}

// EnrollUser enroll 一个已经注册的用户并保存相关信息
// username、networkName、orgName、mspType用于生成保存信息用的路径
// isTLS 是否是用于TLS的证书？
func (cu *CaUser) Enroll(mspClient *msp.Client, isTLS bool) error {
	var err error
	username := cu.GetUsername()

	if isTLS {
		err = mspClient.Enroll(username, msp.WithSecret(cu.Password), msp.WithProfile("tls"))
	} else {
		err = mspClient.Enroll(username, msp.WithSecret(cu.Password))
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

	err = cu.BuildDir(cainfo.CAChain, cert, privkey)
	if err != nil {
		return errors.WithMessage(err, "fail to store info")
	}

	return nil
}
