package model

import (
	"database/sql/driver"
	"fmt"
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

func (peer *Peer) GetTLSCert() string {
	// TODO
	return `
-----BEGIN CERTIFICATE-----
MIICdzCCAh2gAwIBAgIQTtcDS7cbBR3ufL+2KllUljAKBggqhkjOPQQDAjBsMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzEUMBIGA1UEChMLZXhhbXBsZS5jb20xGjAYBgNVBAMTEXRsc2NhLmV4
YW1wbGUuY29tMB4XDTIxMDIyMjExMTUwMFoXDTMxMDIyMDExMTUwMFowWDELMAkG
A1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBGcmFu
Y2lzY28xHDAaBgNVBAMTE29yZGVyZXIuZXhhbXBsZS5jb20wWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAASFRrDq1OeiBerm3MVU8I/w4r71z7oqGDki5g6IFOe0NHmG
SnonawNY4UGW4qCInetTkueQuuVIDEUWecsf7r7Ao4G0MIGxMA4GA1UdDwEB/wQE
AwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwDAYDVR0TAQH/BAIw
ADArBgNVHSMEJDAigCB7WavWZFEVZOvd+x/DlXyvozsyuA+wlmzR3dETVlPCCzBF
BgNVHREEPjA8ghNvcmRlcmVyLmV4YW1wbGUuY29tggdvcmRlcmVyghNvcmRlcmVy
LmV4YW1wbGUuY29tggdvcmRlcmVyMAoGCCqGSM49BAMCA0gAMEUCIQDyTpnIAk+1
oIDMPwWKoo0ntA8ta6JoSs7U0hTn2E7hfAIgO8kAvJnnEPAiTFWdVxCPj5IDPfDV
cQpADaTqJG8cnXU=
-----END CERTIFICATE-----
`
}