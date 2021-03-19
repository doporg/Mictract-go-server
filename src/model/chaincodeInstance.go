package model

import (
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/policydsl"
	"github.com/pkg/errors"

	cb "github.com/hyperledger/fabric-protos-go/common"
)

// Chaincode on the channel
type ChaincodeInstance struct {
	Label	 	 string	`json:"label"`
	ExCC     	 bool	`json:"ex_cc"`
	PolicyStr    string	`json:"policy"`
	Version  	 string `json:"version"`
	Sequence 	 string `json:"sequence"`
	InitRequired bool 	`json:"init_required"`
}

// "OR('Org1MSP.member')"
func NewChaincodeInstance(label, policyStr, version, sequence string, excc, initrequired bool) (*ChaincodeInstance, error) {
	cci := &ChaincodeInstance{
		Label: label,
		ExCC: excc,
		PolicyStr: policyStr,
		Version: version,
		Sequence: sequence,
		InitRequired: initrequired,
	}

	if _, err := cci.GeneratePolicy(); err != nil {
		return &ChaincodeInstance{}, errors.WithMessage(err, "check your policyStr")
	}

	return cci, nil
}

func (cci *ChaincodeInstance)GeneratePolicy() (*cb.SignaturePolicyEnvelope, error) {
	ccPolicy, err := policydsl.FromString(cci.PolicyStr)
	if err != nil {
		return nil, err
	}
	return ccPolicy, nil
}