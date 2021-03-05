package model

import (
	"database/sql/driver"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
)

type Peer struct {
	// Name should be domain name.
	// Example: peer1.org1.net1.com
	Name string `json:"name"`
}

type Peers []Peer

// 自定义数据字段所需实现的两个接口
func (peers *Peers) Scan(value interface{}) error {
	return scan(&peers, value)
}

func (peers *Peers) Value() (driver.Value, error) {
	return value(peers)
}

func (peer *Peer) GetURL() string {
	causer := NewCaUserFromDomainName(peer.Name)
	return fmt.Sprintf("grpcs://peer%d-org%d-net%d:7051", causer.UserID, causer.OrganizationID, causer.NetworkID)
	// return "grpcs://" + strings.ReplaceAll(peer.Name, ".", "-") + ":7051"
}

func (peer *Peer)JoinChannel(channelID, ordererURL string) error {
	user := NewCaUserFromDomainName(peer.Name)
	sdk, err:= GetSDKByNetWorkID(user.NetworkID)
	if err != nil {
		return errors.WithMessage(err, "fail to get sdk ")
	}

	rcp := sdk.Context(fabsdk.WithUser(fmt.Sprintf("Admin@org%d.net%d.com", user.OrganizationID, user.NetworkID)), fabsdk.WithOrg(fmt.Sprintf("org%d", user.OrganizationID)))
	rc, err := resmgmt.New(rcp)
	if err != nil {
		return errors.WithMessage(err, "fail to get rc ")
	}

	return rc.JoinChannel(channelID, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint(ordererURL))
}