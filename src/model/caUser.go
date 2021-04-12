package model

import (
	"fmt"
	"mictract/config"
	"os"
	"path/filepath"
	"strings"

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