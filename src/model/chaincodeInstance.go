package model

import (
	"database/sql/driver"
	"fmt"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/policydsl"
	"github.com/pkg/errors"
	"mictract/config"
	"mictract/global"
	"mictract/model/kubernetes"
	"path/filepath"
)

// Chaincode on the channel
type ChaincodeInstance struct {
	Label	 	 string	`json:"label"`
	ExCC     	 bool	`json:"ex_cc"`
	// If ExCC is false, Adress will be omitted
	Address 	 string	`json:"address"`
	PolicyStr    string	`json:"policy"`
	Version  	 string `json:"version"`
	Sequence 	 int64  `json:"sequence"`
	InitRequired bool 	`json:"init_required"`
	
	PackageID	 string	`json:"package_id"`

	ChannelID	 int	`json:"channel_id"`
	NetworkID	 int	`json:"network_id"`

	// Chaincode.ID
	CCID		 int	`json:"ccid"`
}

type ChaincodeInstances []ChaincodeInstance

// 自定义数据字段所需实现的两个接口
func (ccis *ChaincodeInstances) Scan(value interface{}) error {
	return scan(&ccis, value)
}

func (ccis ChaincodeInstances) Value() (driver.Value, error) {
	return value(ccis)
}

func (cci *ChaincodeInstance) Scan(value interface{}) error {
	return scan(&cci, value)
}

func (cci ChaincodeInstance) Value() (driver.Value, error) {
	return value(cci)
}

func (cci *ChaincodeInstance) getAddress() string {
	return fmt.Sprintf(
		"%s-chaincode%d-channel%d-net%d:9999",
		cci.Label,
		cci.CCID,
		cci.ChannelID,
		cci.NetworkID)
}

// "OR('Org1MSP.member')"
func (cc *Chaincode)NewChaincodeInstance(networkID, channelID int,
	label, address, policyStr, version string,
	sequence int64,
	excc, initrequired bool) (*ChaincodeInstance, error) {

	cci := &ChaincodeInstance{
		Label: label,
		ExCC: excc,
		Address: address,
		PolicyStr: policyStr,
		Version: version,
		Sequence: sequence,
		InitRequired: initrequired,

		ChannelID: channelID,
		NetworkID: networkID,

		CCID: cc.ID,
	}

	if address == "" {
		cci.Address = cci.getAddress()
	}

	if _, err := cci.GeneratePolicy(); err != nil {
		return &ChaincodeInstance{}, errors.WithMessage(err, "check your policyStr")
	}

	c, err := GetChannelFromNets(cci.ChannelID, cci.NetworkID)
	if err != nil {
		return nil, err
	}
	c.Chaincodes = append(c.Chaincodes, *cci)
	UpdateNets(*c)
	// debug：使用安装后得到的链码ID
	// 说明：这种方式获取的外部链码ID与安装后得到的ID不一样
	// 原因：未知
	//var ccPkg []byte
	//var err error
	//if excc {
	//	ccPkg, err = cc.PackageExternalCC(label, address)
	//	if err != nil {
	//		return nil, errors.WithMessage(err, "fail to package external chaincode")
	//	}
	//} else {
	//	ccPkg, err = cc.PackageCC(label)
	//	if err != nil {
	//		return nil, errors.WithMessage(err, "fail to package chaincode")
	//	}
	//}
	//
	//cci.PackageID = lcpackager.ComputePackageID(label, ccPkg)

	return cci, nil
}

func (cci *ChaincodeInstance)GeneratePolicy() (*cb.SignaturePolicyEnvelope, error) {
	ccPolicy, err := policydsl.FromString(cci.PolicyStr)
	if err != nil {
		return nil, err
	}
	return ccPolicy, nil
}

func (cci *ChaincodeInstance)GetCCPkg() ([]byte, error) {
	cc, err := GetChaincodeByID(cci.CCID)
	if err != nil {
		return nil, errors.WithMessage(err, "fail to get chaincode obj from DB")
	}

	if cci.ExCC {
		return cc.PackageExternalCC(cci.Label, cci.Address)
	} else {
		return cc.PackageCC(cci.Label)
	}
}

// If peerURLs are omitted, the chaincode will be installed on
// all peers in the organization specified by orgResMgmt
func (cci *ChaincodeInstance)InstallCC(orgResMgmt *resmgmt.Client, peerURLs ...string) error {
	ccPkg, err := cci.GetCCPkg()
	if err != nil {
		return errors.WithMessage(err, "fail to get ccPkg")
	}

	installCCReq := resmgmt.LifecycleInstallCCRequest{
		Label:   cci.Label,
		Package: ccPkg,
	}

	resps, err := orgResMgmt.LifecycleInstallCC(
		installCCReq,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts),
		resmgmt.WithTargetEndpoints(peerURLs...))
	if err != nil {
		return err
	}

	if len(resps) > 0 && cci.PackageID != resps[0].PackageID {
		cci.PackageID = resps[0].PackageID
	} else if len(resps) <= 0{
		return errors.New("Chaincode installation error")
	}

	c, err := GetChannelFromNets(cci.ChannelID, cci.NetworkID)
	if err != nil {
		return err
	}
	for i, _ := range c.Chaincodes {
		if c.Chaincodes[i].CCID == cci.CCID {
			c.Chaincodes[i] = *cci
		}
	}
	UpdateNets(*c)

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
func (cci *ChaincodeInstance)ApproveCC(orgResMgmt *resmgmt.Client, ordererURL string, peerURLs ...string) error {
	ccPolicy, err := cci.GeneratePolicy()
	if err != nil {
		return err
	}
	approveCCReq := resmgmt.LifecycleApproveCCRequest{
		Name:              cci.Label,
		Version:           cci.Version,
		PackageID:         cci.PackageID,
		Sequence:          cci.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,     // !!
		InitRequired:      cci.InitRequired, // !!
	}

	txnID, err := orgResMgmt.LifecycleApproveCC(
		fmt.Sprintf("channel%d", cci.ChannelID),
		approveCCReq,
		resmgmt.WithOrdererEndpoint(ordererURL),
		resmgmt.WithTargetEndpoints(peerURLs...),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil || txnID == "" {
		return errors.WithMessage(err, "fail to approve chaincode")
	}

	return nil
}

func (cci *ChaincodeInstance)CheckCCCommitReadiness(orgResMgmt *resmgmt.Client, peerURLs ...string) (*map[string]bool, error) {
	global.Logger.Info("Check CC commit readiness...")

	ccPolicy, err := cci.GeneratePolicy()
	if err != nil {
		return nil, err
	}

	req := resmgmt.LifecycleCheckCCCommitReadinessRequest{
		Name:              cci.Label,
		Version:           cci.Version,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		Sequence:          cci.Sequence,
		InitRequired:      cci.InitRequired,
	}
	resp, err := orgResMgmt.LifecycleCheckCCCommitReadiness(
		fmt.Sprintf("channel%d", cci.ChannelID),
		req,
		resmgmt.WithTargetEndpoints(peerURLs...),
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return nil, err
	}

	global.Logger.Info(fmt.Sprintf("%v", resp.Approvals))

	return &resp.Approvals, nil
}

func (cci *ChaincodeInstance)CommitCC(orgResMgmt *resmgmt.Client, ordererUrl string, peerURLs ...string) error {
	global.Logger.Info("commit chaincode...")

	//ccPolicy := policydsl.SignedByAnyMember([]string{"Org1MSP"})
	ccPolicy, err := cci.GeneratePolicy()
	if err != nil {
		return err
	}

	req := resmgmt.LifecycleCommitCCRequest{
		Name:              cci.Label,
		Version:           cci.Version,
		Sequence:          cci.Sequence,
		EndorsementPlugin: "escc",
		ValidationPlugin:  "vscc",
		SignaturePolicy:   ccPolicy,
		InitRequired:      cci.InitRequired,
	}
	txID, err := orgResMgmt.LifecycleCommitCC(
		fmt.Sprintf("channel%d", cci.ChannelID),
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

func (cci *ChaincodeInstance)QueryCommittedCC(orgResMgmt *resmgmt.Client) error {
	req := resmgmt.LifecycleQueryCommittedCCRequest{
		Name: cci.Label,
	}

	resps, err := orgResMgmt.LifecycleQueryCommittedCC(
		fmt.Sprintf("channel%d", cci.ChannelID),
		req,
		resmgmt.WithRetry(retry.DefaultResMgmtOpts))
	if err != nil {
		return err
	}
	global.Logger.Info(fmt.Sprintf("%v", resps))
	for _, resp := range resps {
		if resp.Name == cci.Label {
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
func (cci *ChaincodeInstance)InitCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	if !cci.InitRequired {
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
			ChaincodeID: cci.Label,
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
func (cci *ChaincodeInstance)ExecuteCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Execute(
		channel.Request{
			ChaincodeID: cci.Label,
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
func (cci *ChaincodeInstance)QueryCC(channelClient *channel.Client, args []string, peerURLs ...string) (channel.Response, error) {
	_args := [][]byte{}
	if len(args) < 1{
		return channel.Response{}, errors.New("check your args!")
	} else if len(args) > 1 {
		_args = packArgs(args[1:])
	}

	response, err := channelClient.Query(
		channel.Request{
			ChaincodeID: cci.Label,
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

func (cci *ChaincodeInstance) Build() error {
	cc, err := GetChaincodeByID(cci.CCID)
	if err != nil {
		return err
	}

	if cc.Type != pb.ChaincodeSpec_GOLANG {
		return errors.New("Only supports golang")
	}

	tools := kubernetes.Tools{}
	if _, _, err := tools.ExecCommand(
		filepath.Join(config.LOCAL_SCRIPTS_PATH, "external", "build.sh"),
		filepath.Join(cc.GetCCPath(), "chaincode"),
		filepath.Join(cc.GetCCPath(), "src")); err != nil {
		return err
	}

	return nil
}

func (cci *ChaincodeInstance) CreateEntity() error {
	cc, err := GetChaincodeByID(cci.CCID)
	if err != nil {
		return err
	}

	if cc.Type != pb.ChaincodeSpec_GOLANG {
		return errors.New("Currently only supports golang, please manually start other language chaincodes")
	}

	global.Logger.Info("Starting external chaincode")
	if err := kubernetes.NewChaincode(cci.NetworkID, cci.ChannelID, cci.PackageID, cci.CCID).AwaitableCreate(); err != nil {
		return err
	}
	global.Logger.Info("Successful start of external chaincode")
	return nil
}

func (cci *ChaincodeInstance) RemoveEntity()  {
	global.Logger.Info("Removing external chaincode")
	kubernetes.NewChaincode(cci.NetworkID, cci.ChannelID, cci.PackageID, cci.CCID).Delete()
}