package utils

import (
	"mictract/model"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func SaveCertAndPrivkey(cacert, cert, privkey []byte, isTLS bool, causer model.CaUser) error {
	// 此段代码生成的prefixPath目录下应该只需包括msp和tls两个文件夹
	prefixPath := filepath.Join(GetNetworksMountDirectory(), causer.NetName)
	domainName := causer.NetName + ".com"
	if strings.Contains(causer.OrgName, "orderer") {

		prefixPath = filepath.Join(prefixPath, "ordererOrganizations", domainName)
		if causer.MspType == "orderer" {
			prefixPath = filepath.Join(prefixPath, "orderers", causer.Username)
		} else {
			prefixPath = filepath.Join(prefixPath, "users", causer.Username)
		}
	} else {
		domainName = causer.OrgName + "." + domainName
		prefixPath = filepath.Join(prefixPath, "peerOrganizations", domainName)
		if causer.MspType == "peer" {
			prefixPath = filepath.Join(prefixPath, "peers", causer.Username)
		} else {
			prefixPath = filepath.Join(prefixPath, "users", causer.Username)
		}
	}

	if isTLS {
		prefixPath = filepath.Join(prefixPath, "tls")
		err := os.MkdirAll(prefixPath, os.ModePerm)
		if err != nil {
			return errors.WithMessage(err, prefixPath+"创建错误")
		}

		fuckName := ""
		if causer.MspType == "peer" || causer.MspType == "orderer" {
			fuckName = "server"
		} else {
			fuckName = "client"
		}

		// 写入三个文件 server.crt server.key ca.crt 或者 client.crt client.key ca.crt
		for _, filename := range []string{filepath.Join(prefixPath, fuckName+".crt"),
			filepath.Join(prefixPath, fuckName+".key"),
			filepath.Join(prefixPath, "ca.crt")} {
			f, err := os.Create(filename)
			defer f.Close()
			if err != nil {
				return err
			}

			if strings.HasSuffix(filename, "key") {
				f.Write(privkey)
			} else if strings.HasSuffix(filename, "ca.crt") {
				f.Write(cacert)
			} else {
				f.Write(cert)
			}
		}

	} else {
		prefixPath = filepath.Join(prefixPath, "msp")
		err := os.MkdirAll(prefixPath, os.ModePerm)
		if err != nil {
			return err
		}
		/*
			msp 下有四个文件夹 cacerts tlscacerts keystore signcerts
			tlscacerts 和 cacerts文件夹中的文件夹一样，我们规定一个组织只
			用一个ca
		*/
		for _, dir := range []string{filepath.Join(prefixPath, "cacerts"),
			filepath.Join(prefixPath, "tlscacerts"),
			filepath.Join(prefixPath, "keystore"),
			filepath.Join(prefixPath, "signcerts")} {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}
		}

		certNameSuffix := domainName + "-cert.pem"

		f1, err := os.Create(filepath.Join(prefixPath, "cacerts", "ca."+certNameSuffix))
		defer f1.Close()
		if err != nil {
			return err
		}
		f1.Write(cacert)

		f2, err := os.Create(filepath.Join(prefixPath, "tlscacerts", "tlsca."+certNameSuffix))
		defer f2.Close()
		if err != nil {
			return err
		}
		f2.Write(cacert)

		f3, err := os.Create(filepath.Join(prefixPath, "signcerts", causer.Username+"-cert.com"))
		defer f3.Close()
		if err != nil {
			return err
		}
		f3.Write(cert)

		f4, err := os.Create(filepath.Join(prefixPath, "keystore", "priv_sk"))
		defer f4.Close()
		if err != nil {
			return err
		}
		f4.Write(privkey)

	}
	return nil
}
