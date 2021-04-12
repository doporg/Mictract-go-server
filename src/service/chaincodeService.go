package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/policydsl"
	"github.com/pkg/errors"
	"mictract/config"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"path/filepath"

	lcpackager "github.com/hyperledger/fabric-sdk-go/pkg/fab/ccpackager/lifecycle"
)

type ChaincodeService struct {
	cc *model.Chaincode
}

func NewChaincodeService(cc *model.Chaincode) *ChaincodeService {
	return &ChaincodeService{
		cc: cc,
	}
}

// 本地打包和安装后生成的不同 why？
func (ccSvc *ChaincodeService)GetPackageID(label string, ccPkg []byte) string {
	return lcpackager.ComputePackageID(label, ccPkg)
}

func (ccSvc *ChaincodeService)GeneratePolicy(policyStr string) (*cb.SignaturePolicyEnvelope, error) {
	return policydsl.FromString(policyStr)
}

func (ccSvc *ChaincodeService)GetCCPkg() ([]byte, error) {
	return ccSvc.PackageExternalCC(ccSvc.cc.Label, ccSvc.cc.GetAddress())
}

func (ccSvc *ChaincodeService)Unpack() error {
	// tar zxvf src.tar.gz
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		"tar",
		"zxvf",
		filepath.Join(ccSvc.cc.GetCCPath(), "src.tar.gz"),
		"-C",
		ccSvc.cc.GetCCPath()); err != nil {
		dao.UpdateChaincodeStatusByID(ccSvc.cc.ID, enum.StatusError)
		return err
	}

	ccPkg, err := ccSvc.GetCCPkg()
	if err != nil {
		return err
	}
	packageID := ccSvc.GetPackageID(ccSvc.cc.Label, ccPkg)
	if err := dao.UpdateChaincodePackageIDByID(ccSvc.cc.ID, packageID); err != nil {
		return err
	}
	global.Logger.Info("├── test_local_packageID: " + packageID)

	return nil
}

func (ccSvc *ChaincodeService)PackageExternalCC(label, address string) (ccPkg []byte, err error) {
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

// If peerURLs are omitted, the chaincode will be installed on
// all peers in the organization specified by orgResMgmt
func (ccSvc *ChaincodeService)InstallCC(orgResMgmt *resmgmt.Client, peerURLs ...string) error {
	ccPkg, err := ccSvc.GetCCPkg()
	if err != nil {
		return errors.WithMessage(err, "fail to get ccPkg")
	}

	installCCReq := resmgmt.LifecycleInstallCCRequest{
		Label:   ccSvc.cc.Label,
		Package: ccPkg,
	}

	resps, err := orgResMgmt.LifecycleInstallCC(
		installCCReq,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithTargetEndpoints(peerURLs...))
	if err != nil {
		return err
	}

	if len(resps) > 0 && ccSvc.cc.PackageID != resps[0].PackageID {
		ccSvc.cc.PackageID = resps[0].PackageID
	} else if len(resps) <= 0{
		return errors.New("Chaincode installation error")
	}

	global.Logger.Info("chaincode installed successfully")
	for _, resp := range resps {
		global.Logger.Info("├── test_local_packageID: " + ccSvc.GetPackageID(ccSvc.cc.Label, ccPkg))
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
func (ccSvc *ChaincodeService)ApproveCC(orgResMgmt *resmgmt.Client, ordererURL string, peerURLs ...string) error {
	ccPolicy, err := ccSvc.GeneratePolicy(ccSvc.cc.PolicyStr)
	if err != nil {
		return err
	}
	approveCCReq := resmgmt.LifecycleApproveCCRequest{
		Name:              ccSvc.cc.GetName(),
		Version:           ccSvc.cc.Version,
		PackageID:         ccSvc.cc.PackageID,
		Sequence:          ccSvc.cc.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,     // !!
		InitRequired:      ccSvc.cc.InitRequired, // !!
	}

	txnID, err := orgResMgmt.LifecycleApproveCC(
		model.GetChannelNameByID(ccSvc.cc.ChannelID),
		approveCCReq,
		resmgmt.WithOrdererEndpoint(ordererURL),
		resmgmt.WithTargetEndpoints(peerURLs...),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil || txnID == "" {
		return errors.WithMessage(err, "fail to approve chaincode")
	}

	return nil
}

func (ccSvc *ChaincodeService)CheckCCCommitReadiness(orgResMgmt *resmgmt.Client, peerURLs ...string) (*map[string]bool, error) {

	ccPolicy, err := ccSvc.GeneratePolicy(ccSvc.cc.PolicyStr)
	if err != nil {
		return &map[string]bool{}, err
	}

	ch, err := dao.FindChannelByID(ccSvc.cc.ChannelID)
	if err != nil {
		return &map[string]bool{}, err
	}

	req := resmgmt.LifecycleCheckCCCommitReadinessRequest{
		Name:              ccSvc.cc.GetName(),
		Version:           ccSvc.cc.Version,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		Sequence:          ccSvc.cc.Sequence,
		InitRequired:      ccSvc.cc.InitRequired,
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

func (ccSvc *ChaincodeService)CommitCC(orgResMgmt *resmgmt.Client, ordererUrl string, peerURLs ...string) error {
	//ccPolicy := policydsl.SignedByAnyMember([]string{"Org1MSP"})
	ccPolicy, err := ccSvc.GeneratePolicy(ccSvc.cc.PolicyStr)
	if err != nil {
		return err
	}

	req := resmgmt.LifecycleCommitCCRequest{
		Name:              ccSvc.cc.GetName(),
		Version:           ccSvc.cc.Version,
		Sequence:          ccSvc.cc.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		InitRequired:      ccSvc.cc.InitRequired,
	}
	txID, err := orgResMgmt.LifecycleCommitCC(
		model.GetChannelNameByID(ccSvc.cc.ChannelID),
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

func (ccSvc *ChaincodeService)QueryCommittedCC(orgResMgmt *resmgmt.Client) error {
	req := resmgmt.LifecycleQueryCommittedCCRequest{
		Name: ccSvc.cc.GetName(),
	}

	resps, err := orgResMgmt.LifecycleQueryCommittedCC(
		model.GetChannelNameByID(ccSvc.cc.ChannelID),
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return err
	}
	global.Logger.Info(fmt.Sprintf("%v", resps))
	for _, resp := range resps {
		if resp.Name == ccSvc.cc.Label {
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
func (ccSvc *ChaincodeService)InitCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	if !ccSvc.cc.InitRequired {
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
			ChaincodeID: ccSvc.cc.GetName(),
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
func (ccSvc *ChaincodeService)ExecuteCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: ccSvc.cc.GetName(),
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
func (ccSvc *ChaincodeService)QueryCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Query(
		channel.Request{
			ChaincodeID: ccSvc.cc.GetName(),
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

func (ccSvc *ChaincodeService) Build() error {
	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "external", "build.sh"),
		filepath.Join(ccSvc.cc.GetCCPath(), "chaincode"),
		filepath.Join(ccSvc.cc.GetCCPath(), "src")); err != nil {
		return err
	}

	return nil
}

func (ccSvc *ChaincodeService) CreateEntity() error {
	global.Logger.Info("Starting external chaincode")
	if err := kubernetes.
		NewChaincode(ccSvc.cc.NetworkID, ccSvc.cc.ChannelID, ccSvc.cc.PackageID, ccSvc.cc.ID).
		AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("Successful start of external chaincode")
	return nil
}

func (ccSvc *ChaincodeService) RemoveEntity()  {
	global.Logger.Info("Removing external chaincode")
	kubernetes.NewChaincode(ccSvc.cc.NetworkID, ccSvc.cc.ChannelID, ccSvc.cc.PackageID, ccSvc.cc.ID).Delete()
}
