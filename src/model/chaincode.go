package model

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	cb "github.com/hyperledger/fabric-protos-go/common"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	lcpackager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/lifecycle"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/policydsl"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"mictract/config"
	"mictract/enum"
	"mictract/global"
	"mictract/model/kubernetes"
	"os"
	"path/filepath"
)

// Local chaincode
type Chaincode struct {
	ID  			int		                    `json:"id" gorm:"primarykey"`
	Nickname		string	                    `json:"nickname"`
	Status 			string						`json:"status"`

	ChannelID	 	int							`json:"channel_id"`
	NetworkID	 	int							`json:"network_id"`

	Label	 	 	string						`json:"label"`
	PolicyStr    	string						`json:"policy"`
	Version  	 	string 						`json:"version"`
	Sequence 	 	int64  						`json:"sequence"`
	InitRequired 	bool 						`json:"init_required"`

	PackageID	 	string						`json:"package_id"`
}

// tar czf src.tar.gz src
func NewChaincode(nickname string, chID, netID int, label, policyStr, version string, seq int64, initReq bool) (*Chaincode, error){
	// 1. check
	net, _ := FindNetworkByID(netID)
	if net.Status != enum.StatusRunning {
		return &Chaincode{}, errors.New("Unable to create chaincode, please check network status")
	}
	ch, _ := FindChannelByID(chID)
	if ch.Status != enum.StatusRunning {
		return &Chaincode{}, errors.New("Unable to create chaincode, please check channel status")
	}

	cc := Chaincode{
		Nickname: nickname,
		Status: enum.StatusUnpacking,
		ChannelID: chID,
		NetworkID: netID,

		Label: label,
		PolicyStr: policyStr,
		Version: version,
		Sequence: seq,
		InitRequired: initReq,
	}

	if err := global.DB.Create(&cc).Error; err != nil {
		return &Chaincode{}, err
	}

	//cc.ID
	// mkdir chaincodes/chaincodeID
	if err := os.MkdirAll(cc.GetCCPath(), os.ModePerm); err != nil {
		return &Chaincode{}, err
	}

	return &cc, nil
}

func (c *Chaincode) GetName() string {
	return fmt.Sprintf("chaincode%d", c.ID)
}

func (c *Chaincode) GetAddress() string {
	return fmt.Sprintf(
		"%s-%s-channel%d-net%d:9999",
		c.Label,
		c.GetName(),
		c.ChannelID,
		c.NetworkID)
}

func (c *Chaincode)GetCCPath() string {
	return filepath.Join(
		config.LOCAL_CC_PATH,
		fmt.Sprintf("chaincode%d", c.ID))
}

func (cc *Chaincode)Unpack() error {
	// tar zxvf src.tar.gz
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		"tar",
		"zxvf",
		filepath.Join(cc.GetCCPath(), "src.tar.gz"),
		"-C",
		cc.GetCCPath()); err != nil {
		global.Logger.Error("fail to unpack cc ", zap.Error(err))
		cc.UpdateStatus(enum.StatusError)
		return err
	}
	return nil
}

func FindChaincodeByID(ccID int) (*Chaincode, error) {
	var ccs []Chaincode
	if err := global.DB.Where("id = ?", ccID).Find(&ccs).Error; err != nil {
		return &Chaincode{}, err
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

func UpdateChaincodeNickname(ccID int, newNickname string) error {
	if err := global.DB.Model(&Chaincode{}).
		Where("id = ?", ccID).
		Updates(Chaincode{Nickname: newNickname}).Error; err != nil {
		return errors.WithMessage(err, "Fail to update")
	}
	return nil
}

func (cc *Chaincode)UpdateStatus(status string) error {
	return global.DB.Model(&Chaincode{}).Where("id = ?", cc.ID).Update("status", status).Error
}

func (cc *Chaincode)PackageCC(ccLabel string) (ccPkg []byte, err error) {
	ccSrcPath := filepath.Join(
		cc.GetCCPath(),
		"src")
	desc := &lcpackager.Descriptor{
		Path:  ccSrcPath,
		Type:  pb.ChaincodeSpec_GOLANG,
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
		return []byte{}, errors.WithMessage(err, "fail to generate connection.json")
	}

	err = tw1.Close()
	if err == nil {
		err = gw1.Close()
	}
	if err != nil {
		return []byte{}, err
	}

	content = []byte(`{"path":"","type":"external","label":"` + label + `"}`)
	payload2 := bytes.NewBuffer(nil)
	gw2 := gzip.NewWriter(payload2)
	tw2 := tar.NewWriter(gw2)

	if err := writePackage(tw2, "code.tar.gz", payload1.Bytes()); err != nil {
		return []byte{}, errors.WithMessage(err, "fail to generate code.tar.gz")
	}
	if err := writePackage(tw2, "metadata.json", content); err != nil {
		return []byte{}, errors.WithMessage(err, "fail to generate metadata.json")
	}

	err = tw2.Close()
	if err == nil {
		err = gw2.Close()
	}
	if err != nil {
		return []byte{}, err
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


func (cc *Chaincode)GeneratePolicy() (*cb.SignaturePolicyEnvelope, error) {
	ccPolicy, err := policydsl.FromString(cc.PolicyStr)
	if err != nil {
		return &cb.SignaturePolicyEnvelope{}, err
	}
	return ccPolicy, nil
}

func (cc *Chaincode)GetCCPkg() ([]byte, error) {
	return cc.PackageExternalCC(cc.Label, cc.GetAddress())
}

// If peerURLs are omitted, the chaincode will be installed on
// all peers in the organization specified by orgResMgmt
func (cc *Chaincode)InstallCC(orgResMgmt *resmgmt.Client, peerURLs ...string) error {
	ccPkg, err := cc.GetCCPkg()
	if err != nil {
		return errors.WithMessage(err, "fail to get ccPkg")
	}

	installCCReq := resmgmt.LifecycleInstallCCRequest{
		Label:   cc.Label,
		Package: ccPkg,
	}

	resps, err := orgResMgmt.LifecycleInstallCC(
		installCCReq,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithTargetEndpoints(peerURLs...))
	if err != nil {
		return err
	}

	if len(resps) > 0 && cc.PackageID != resps[0].PackageID {
		cc.PackageID = resps[0].PackageID
	} else if len(resps) <= 0{
		return errors.New("Chaincode installation error")
	}

	global.Logger.Info("chaincode installed successfully")
	for _, resp := range resps {
		global.Logger.Info("├── target: " + resp.Target)
		global.Logger.Info(fmt.Sprintf("├── status: %d", resp.Status))
		global.Logger.Info("└── packageID: " + resp.PackageID)
	}

	return nil
}

/*
peer lifecycle chaincode checkcommitreadiness --channelID mychannel --name marriage --version 1 --sequence 1 --init-required --signature-policy "OR('Org1MSP.member')"
批准链码时指定的这些信息共同标识了链码，查询的时候少了是查不出来的，比如批准的时候InitRequired = true
查询的时候没有--init-required，查不出来；批准的时候指定了策略，查询的时候没有指定策略或者指定的和开始
不一样，都查不出来
*/
func (cc *Chaincode)ApproveCC(orgResMgmt *resmgmt.Client, ordererURL string, peerURLs ...string) error {
	ccPolicy, err := cc.GeneratePolicy()
	if err != nil {
		return err
	}
	approveCCReq := resmgmt.LifecycleApproveCCRequest{
		Name:              cc.Label,
		Version:           cc.Version,
		PackageID:         cc.PackageID,
		Sequence:          cc.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,     // !!
		InitRequired:      cc.InitRequired, // !!
	}

	txnID, err := orgResMgmt.LifecycleApproveCC(
		fmt.Sprintf("channel%d", cc.ChannelID),
		approveCCReq,
		resmgmt.WithOrdererEndpoint(ordererURL),
		resmgmt.WithTargetEndpoints(peerURLs...),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil || txnID == "" {
		return errors.WithMessage(err, "fail to approve chaincode")
	}

	return nil
}

func (cc *Chaincode)CheckCCCommitReadiness(orgResMgmt *resmgmt.Client, peerURLs ...string) (*map[string]bool, error) {
	global.Logger.Info("Check CC commit readiness...")

	ccPolicy, err := cc.GeneratePolicy()
	if err != nil {
		return &map[string]bool{}, err
	}

	ch, err := FindChannelByID(cc.ChannelID)
	if err != nil {
		return &map[string]bool{}, err
	}

	req := resmgmt.LifecycleCheckCCCommitReadinessRequest{
		Name:              cc.Label,
		Version:           cc.Version,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		Sequence:          cc.Sequence,
		InitRequired:      cc.InitRequired,
	}
	resp, err := orgResMgmt.LifecycleCheckCCCommitReadiness(
		ch.GetName(),
		req,
		resmgmt.WithTargetEndpoints(peerURLs...),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return &map[string]bool{}, err
	}

	return &resp.Approvals, nil
}

func (cc *Chaincode)CommitCC(orgResMgmt *resmgmt.Client, ordererUrl string, peerURLs ...string) error {
	global.Logger.Info("commit chaincode...")

	//ccPolicy := policydsl.SignedByAnyMember([]string{"Org1MSP"})
	ccPolicy, err := cc.GeneratePolicy()
	if err != nil {
		return err
	}

	req := resmgmt.LifecycleCommitCCRequest{
		Name:              cc.Label,
		Version:           cc.Version,
		Sequence:          cc.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		InitRequired:      cc.InitRequired,
	}
	txID, err := orgResMgmt.LifecycleCommitCC(
		fmt.Sprintf("channel%d", cc.ChannelID),
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithOrdererEndpoint(ordererUrl),
		resmgmt.WithTargetEndpoints(peerURLs...))
	if err != nil {
		return err
	}

	global.Logger.Info("txID: " + string(txID))

	return nil
}

func (cc *Chaincode)QueryCommittedCC(orgResMgmt *resmgmt.Client) error {
	req := resmgmt.LifecycleQueryCommittedCCRequest{
		Name: cc.Label,
	}

	resps, err := orgResMgmt.LifecycleQueryCommittedCC(
		fmt.Sprintf("channel%d", cc.ChannelID),
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return err
	}
	global.Logger.Info(fmt.Sprintf("%v", resps))
	for _, resp := range resps {
		if resp.Name == cc.Label {
			return nil
		}
	}
	return errors.New("Did't find the submitted cc")
}

// []string to [][]byte

func packArgs(paras []string) [][]byte {
	var args [][]byte
	for _, k := range paras {
		args = append(args, []byte(k))
	}
	return args
}


// shell批准时指定--init-required，或者sdk批准时指定 InitRequired = true，
// 运行链码时都需要先初始化链码，用--isInit或者IsInit: true
func (cc *Chaincode)InitCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	if !cc.InitRequired {
		return channel.Response{}, errors.New("This chaincode does not need init")
	}
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}
	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: cc.Label,
			Fcn: args[0],
			Args: _args,
			IsInit: true},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(peerURLs...),
	)
	if err != nil {
		return response, errors.WithMessage(err, "fail to init chaincode")
	}
	return response, err
}

// If you do not specify peerURLs,
// the program seems to automatically find peers that meet the policy to endorse.
// If specified,
// you must be responsible for satisfying the endorsement strategy
func (cc *Chaincode)ExecuteCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: cc.Label,
			Fcn: args[0],
			Args: _args},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(peerURLs...),
	)
	if err != nil {
		return channel.Response{}, errors.WithMessage(err, "fail to execute chaincode！")
	}

	return response, err
}

// eg: QueryCC(cc, "mycc", []string{"Query", "a"}, "peer0.org1.example.com")
func (cc *Chaincode)QueryCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Query(
		channel.Request{
			ChaincodeID: cc.Label,
			Fcn: args[0],
			Args: _args},
		channel.WithRetry(retry.DefaultChannelOpts),
		channel.WithTargetEndpoints(peerURLs...),
	)
	if err != nil {
		return channel.Response{}, errors.WithMessage(err, "fail to execute qeury！")
	}
	return response, nil
}

func (cc *Chaincode) Build() error {
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "external", "build.sh"),
		filepath.Join(cc.GetCCPath(), "chaincode"),
		filepath.Join(cc.GetCCPath(), "src")); err != nil {
		return err
	}

	return nil
}

func (cc *Chaincode) CreateEntity() error {
	global.Logger.Info("Starting external chaincode")
	if err := kubernetes.NewChaincode(cc.NetworkID, cc.ChannelID, cc.PackageID, cc.ID).AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("Successful start of external chaincode")
	return nil
}

func (cc *Chaincode) RemoveEntity()  {
	global.Logger.Info("Removing external chaincode")
	kubernetes.NewChaincode(cc.NetworkID, cc.ChannelID, cc.PackageID, cc.ID).Delete()
}