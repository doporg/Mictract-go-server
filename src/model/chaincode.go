package model

import (
	"encoding/json"
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
		config.LOCAL_CC_PATH,
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
	// generate connection.json
	connection := []byte(`{
		"address": "` + address + `",
		"dial_timeout": "10s",
		"tls_required": false,
		"client_auth_required": false,
		"client_key": "-----BEGIN EC PRIVATE KEY----- ... -----END EC PRIVATE KEY-----",
		"client_cert": "-----BEGIN CERTIFICATE----- ... -----END CERTIFICATE-----",
		"root_cert": "-----BEGIN CERTIFICATE---- ... -----END CERTIFICATE-----"
	}`)

	f, err := os.Create(filepath.Join(cc.GetCCPath(), "connection.json"))
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(connection); err != nil {
		return nil, err
	}
	f.Close()

	// generate code.tar.gz by connection.json
	// tar cfz code.tar.gz connection.json
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand("tar",
		"cfz",
		filepath.Join(cc.GetCCPath(), "code.tar.gz"),
		filepath.Join(cc.GetCCPath(), "connection.json")); err != nil {
		return nil, err
	}

	// generate metadata.json
	metadata, err := json.Marshal(lcpackager.PackageMetadata{
		Path: "Marx bless, no bugs",
		Type: "external",
		Label: label,
	})
	if err != nil {
		return nil, err
	}
	f, err = os.Create(filepath.Join(cc.GetCCPath(), "metadata.json"))
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(metadata); err != nil {
		return nil, err
	}
	f.Close()

	// generate external package
	// tar cfz label.tgz metadata.json code.tar.gz
	if _, _, err := tools.ExecCommand("tar", "cfz",
		filepath.Join(cc.GetCCPath(), fmt.Sprintf("%s.tgz", label)),
		filepath.Join(cc.GetCCPath(), "metadata.json"),
		filepath.Join(cc.GetCCPath(), "code.tar.gz")); err != nil {
		return nil, err
	}

	f, err = os.Open(filepath.Join(cc.GetCCPath(), fmt.Sprintf("%s.tgz", label)))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Read(ccPkg); err != nil {
		return nil, err
	}
	return ccPkg, nil
}