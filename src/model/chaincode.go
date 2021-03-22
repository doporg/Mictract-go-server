package model

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"mictract/config"
	"mictract/global"
	"mictract/model/kubernetes"
	"os"
	"path/filepath"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	lcpackager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/lifecycle"
)

// Local chaincode
type Chaincode struct {
	ID  		int		                    `json:"id" gorm:"primarykey"`
	Type        pb.ChaincodeSpec_Type		`json:"type"`
	// temp
	Nickname	string	                    `json:"nickname"`
}

func (c *Chaincode)GetCCPath() string {
	return filepath.Join(
		config.LOCAL_CC_PATH,
		fmt.Sprintf("chaincode%d", c.ID))
}

//
// tar czf src.tar.gz src
func NewChaincode(codeTarGz []byte, nickname string, ccType string) (*Chaincode, error){
	_ccType := pb.ChaincodeSpec_UNDEFINED
	switch ccType {
	case "go", "Go", "golang", "Golang":
		_ccType = pb.ChaincodeSpec_GOLANG
	case "node", "Node", "node.js", "Node.js":
		_ccType = pb.ChaincodeSpec_NODE
	case "java", "Java":
		_ccType = pb.ChaincodeSpec_JAVA
	default:
		return &Chaincode{}, errors.New("The language chain code is not supported")
	}
	cc := Chaincode{
		Nickname: nickname,
		Type: _ccType,
	}

	if err := global.DB.Create(&cc).Error; err != nil {
		return &Chaincode{}, err
	}

	//cc.ID
	// mkdir chaincodes/chaincodeID
	if err := os.MkdirAll(cc.GetCCPath(), os.ModePerm); err != nil {
		return &Chaincode{}, err
	}

	f, err := os.Create(filepath.Join(cc.GetCCPath(), "src.tar.gz"));
	if  err != nil {
		return &Chaincode{}, err
	}
	defer f.Close()

	if _, err := f.Write(codeTarGz); err != nil {
		return &Chaincode{}, err
	}

	// tar zxvf src.tar.gz
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		"tar",
		"zxvf",
		filepath.Join(cc.GetCCPath(), "src.tar.gz"),
		"-C",
		cc.GetCCPath()); err != nil {
		return &Chaincode{}, err
	}

	return &Chaincode{}, nil
}

func GetChaincodeByID(ccID int) (*Chaincode, error) {
	ccs := []Chaincode{}
	if err := global.DB.Where("id = ?", ccID).Find(&ccs).Error; err != nil {
		return &Chaincode{}, err
	}
	if len(ccs) != 1 {
		return &Chaincode{}, errors.New("chaincode not found")
	}
	return &ccs[0], nil
}

func DeleteChaincodeByID(ccID int) error {
	if err := global.DB.Where("id = ?", ccID).Delete(&Chaincode{}).Error; err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(config.LOCAL_CC_PATH, fmt.Sprintf("chaincode%d", ccID))); err != nil {
		return err
	}
	return nil
}

func ListAllChaincodes() ([]Chaincode, error) {
	ccs := []Chaincode{}
	if err := global.DB.Find(&ccs).Error; err != nil {
		return []Chaincode{}, err
	}
	return ccs, nil
}

func (cc *Chaincode)PackageCC(ccLabel string) (ccPkg []byte, err error) {
	ccSrcPath := filepath.Join(
		cc.GetCCPath(),
		"src")
	desc := &lcpackager.Descriptor{
		Path:  ccSrcPath,
		Type:  cc.Type,
		Label: ccLabel,
	}
	ccPkg, err = lcpackager.NewCCPackage(desc)
	return ccPkg, err
}

func (cc *Chaincode)PackageExternalCC(label, address string) (ccPkg []byte, err error) {
	payload1 := bytes.NewBuffer(nil)
	gw1 := gzip.NewWriter(payload1)
	tw1 := tar.NewWriter(gw1)
	content := []byte(`{
		"address": "` + address + `",
		"dial_timeout": "10s",
		"tls_required": false,
		"client_auth_required": false,
		"client_key": "-----BEGIN EC PRIVATE KEY----- ... -----END EC PRIVATE KEY-----",
		"client_cert": "-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----",
		"root_cert": "-----BEGIN CERTIFICATE---- ... -----END CERTIFICATE-----"
	}`)

	err = writePackage(tw1, "connection.json", content)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to generate connection.json")
	}

	err = tw1.Close()
	if err == nil {
		err = gw1.Close()
	}
	if err != nil {
		return nil, err
	}

	content = []byte(`{"path":"","type":"external","label":"` + label + `"}`)
	payload2 := bytes.NewBuffer(nil)
	gw2 := gzip.NewWriter(payload2)
	tw2 := tar.NewWriter(gw2)

	if err := writePackage(tw2, "code.tar.gz", payload1.Bytes()); err != nil {
		return nil, errors.WithMessage(err, "fail to generate code.tar.gz")
	}
	if err := writePackage(tw2, "metadata.json", content); err != nil {
		return nil, errors.WithMessage(err, "fail to generate metadata.json")
	}

	err = tw2.Close()
	if err == nil {
		err = gw2.Close()
	}
	if err != nil {
		return nil, err
	}

	return payload2.Bytes(), nil
}
func writePackage(tw *tar.Writer, name string, payload []byte) error {
	err := tw.WriteHeader(
		&tar.Header{
			Name: name,
			Size: int64(len(payload)),
			Mode: 0100644,
		},
	)
	if err != nil {
		return err
	}

	_, err = tw.Write(payload)
	return err
}