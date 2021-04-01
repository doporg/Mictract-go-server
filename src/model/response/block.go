package response

import (
	"encoding/base64"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
)

type BlockHeightInfo struct {
	Height 				uint64		`json:"height"`
	CurrentBlockHash	string      `json:"currentBlockHash"`
	PreviousBlockHash   string      `json:"previousBlockHash"`
}

func NewBlockHeightInfo(bci *fab.BlockchainInfoResponse) *BlockHeightInfo {
	return &BlockHeightInfo{
		Height: bci.BCI.Height,
		CurrentBlockHash: base64.StdEncoding.EncodeToString(bci.BCI.CurrentBlockHash),
		PreviousBlockHash: base64.StdEncoding.EncodeToString(bci.BCI.PreviousBlockHash),
	}
}

type BlockInfo struct {

}