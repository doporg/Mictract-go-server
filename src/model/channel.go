package model

import (
	"database/sql/driver"
	"encoding/base64"
	"mictract/config"
	"mictract/global"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/msp"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
)

type Channel struct {
	Name          string        `json:"name"`
	NetworkName   string        `json:"networkname"`
	Organizations Organizations `json:"organizations"`
	Orderers      Orders        `json:"orderers"`
}

type Channels []Channel

// 自定义数据字段所需实现的两个接口
func (channels *Channels) Scan(value interface{}) error {
	return scan(&channels, value)
}

func (channels *Channels) Value() (driver.Value, error) {
	return value(channels)
}

func (c *Channel) NewLedgerClient(username, orgname string) (*ledger.Client, error) {
	sdk, ok := global.SDKs[c.NetworkName]
	if !ok {
		return nil, errors.New("fail to get sdk. please update global.SDKs.")
	}
	ledgerClient, err := ledger.New(sdk.ChannelContext(c.Name, fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return nil, err
	}
	return ledgerClient, nil
}

func (c *Channel) NewResmgmtClient(username, orgname string) (*resmgmt.Client, error) {
	sdk, ok := global.SDKs[c.NetworkName]
	if !ok {
		return nil, errors.New("fail to get sdk. please update global.SDKs.")
	}
	resmgmtClient, err := resmgmt.New(sdk.Context(fabsdk.WithUser(username), fabsdk.WithOrg(orgname)))
	if err != nil {
		return nil, err
	}
	return resmgmtClient, nil
}

func (c *Channel)getAndStoreConfig() error {
	if len(c.Organizations) < 1 || len(c.Orderers) < 1 {
		return errors.New("There is no organization in the channel.")
	}

	ledgerClient, err := c.NewLedgerClient("Admin", c.Organizations[0].Name)
	if err != nil {
		return errors.WithMessage(err, "fail to get ledgerClient")
	}

	cfg, err := ledgerClient.QueryConfigBlock()
	if err != nil {
		return errors.WithMessage(err, "fail to query config")
	}

	bt, err := proto.Marshal(cfg)
	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "config_block.pb"))
	if err != nil {
		return err
	}
	_, err = f.Write(bt)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func (c *Channel)updateConfig(signs []msp.SigningIdentity) error {
	envelopeFile, err := os.Open(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "org_update_in_envelope.pb"))
	if err != nil {
		return err
	}

	req := resmgmt.SaveChannelRequest{
		ChannelID:         c.Name,
		ChannelConfig:     envelopeFile,
		SigningIdentities: signs,
	}
	resmgmtClient, err := c.NewResmgmtClient("Admin", c.Organizations[0].Name)
	_, err = resmgmtClient.SaveChannel(req, resmgmt.WithRetry(retry.DefaultResMgmtOpts), resmgmt.WithOrdererEndpoint(c.Orderers[0].Name))
	if err != nil {
		return errors.WithMessage(err, "fail to update channel config")
	}
	return nil
}

func (c *Channel) AddOrg(org *Organization) error {
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	c.Organizations = append(c.Organizations, *org)

	// generate configtx.yaml
	configtxFile, err := os.Create(filepath.Join(config.LOCAL_BASE_PATH, "scripts", "addorg", "configtx.yaml"))
	if err != nil {
		return errors.WithMessage(err, "fail to open configtx.yaml")
	}

	_, err = configtxFile.WriteString(org.GetConfigtxFile())
	if err != nil {
		return errors.WithMessage(err, "fail to write configtx.yaml")
	}

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "addOrg", c.Name, org.MSPID)
	output, err := cmd.CombinedOutput()
	global.Logger.Info(string(output))
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	for _, org := range c.Organizations {
		mspClient, err := org.NewMspClient()
		if err != nil {
			return errors.WithMessage(err, "fail to get mspClient "+org.Name)
		}
		adminIdentity, err := mspClient.GetSigningIdentity("Admin")
		if err != nil {
			return errors.WithMessage(err, org.Name+"fail to sign")
		}
		signs = append(signs, adminIdentity)
	}

	// update org_update_in_envelope.pb
	return c.updateConfig(signs)
}

func (c *Channel)UpdateAnchors(org *Organization) error {
	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	// generate anchors.json
	st := `"{"mod_policy":"Admins","value":{"anchor_peers":[`
	for _, peer := range org.Peers {
		st += `{"host":"` + peer.Name + `",port:7051},`
	}
	st += `]},"version":"0"}"`
	f, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "anchors.json"))
	if err != nil {
		return err
	}
	if _, err = f.WriteString(st); err != nil {
		return err
	}
	f.Close()

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "updateAnchors", c.Name, org.MSPID)
	output, err := cmd.CombinedOutput()
	global.Logger.Info(string(output))
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	for _, org := range c.Organizations {
		mspClient, err := org.NewMspClient()
		if err != nil {
			return errors.WithMessage(err, "fail to get mspClient "+org.Name)
		}
		adminIdentity, err := mspClient.GetSigningIdentity("Admin")
		if err != nil {
			return errors.WithMessage(err, org.Name+"fail to sign")
		}
		signs = append(signs, adminIdentity)
	}

	// update org_update_in_envelope.pb
	return c.updateConfig(signs)

}

// AddOrderers
func (c *Channel)AddOrderers(org *Organization) error {
	if c.Name != "system-channel" {
		return errors.New("only for system-channel")
	}

	// generate config_block.pb
	if err := c.getAndStoreConfig(); err != nil {
		return err
	}

	// generate ord1.json
	st := `["`
	for _, orderer := range org.Peers {
		st += `{"client_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(orderer.GetTLSCert())) +
			`","host":"` + orderer.Name +
			`","port":7050,` +
			`"server_tls_cert":"` + base64.StdEncoding.EncodeToString([]byte(orderer.GetTLSCert())) + `"},`
	}
	st += "]"
	f1, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "ord1.json"))
	if err != nil {
		return err
	}
	if _, err = f1.WriteString(st); err != nil {
		return err
	}
	f1.Close()

	// generate ord2.json
	st = `[`
	for _, orderer := range org.Peers {
		st += `"` + orderer.Name + `",`
	}
	st += "]"
	f2, err := os.Create(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "ord2.json"))
	if err != nil {
		return err
	}
	if _, err = f2.WriteString(st); err != nil {
		return err
	}
	f2.Close()

	// call addorg.sh to generate org_update_in_envelope.pb
	// TODO
	cmd := exec.Command(filepath.Join(config.LOCAL_SCRIPTS_PATH, "addorg", "addorg.sh"), "addOrderers", c.Name)
	output, err := cmd.CombinedOutput()
	global.Logger.Info(string(output))
	if err != nil {
		return errors.WithMessage(err, "fail to exec addorg.sh")
	}

	// sign for org_update_in_envelope.pb and update it
	signs := []msp.SigningIdentity{}
	mspClient, err := org.NewMspClient()
	if err != nil {
		return errors.WithMessage(err, "fail to get mspClient "+org.Name)
	}
	adminIdentity, err := mspClient.GetSigningIdentity("Admin")
	if err != nil {
		return errors.WithMessage(err, org.Name+"fail to sign")
	}
	signs = append(signs, adminIdentity)

	// update org_update_in_envelope.pb
	return c.updateConfig(signs)
}