package response
import (
	"github.com/hyperledger/fabric-protos-go/msp"

	"github.com/hyperledger/fabric-protos-go/common"
	putil "github.com/hyperledger/fabric/protoutil"
)

type Block struct {
	RawBlock			*common.Block	`json:"rawBlock"`
	Data                []Recode		`json:"data"`
}

type Recode struct {
	ChannelHeader 	*common.ChannelHeader
	MSPID 			string
	Creator 		string
}

func ParseBlock(block *common.Block) (*Block, error) {
	var err error
	retBlock := &Block{
		RawBlock: 	block,
		Data: 		[]Recode{},
	}

	for _, envBytes := range block.Data.Data {
		recode := Recode{}

		var env *common.Envelope
		if env, err = putil.GetEnvelopeFromBlock(envBytes); err != nil {
			return &Block{}, err
		}

		var payload *common.Payload
		if payload, err = putil.UnmarshalPayload(env.Payload); err != nil {
			return &Block{}, err
		}

		var chdr *common.ChannelHeader
		if chdr, err = putil.UnmarshalChannelHeader(payload.Header.ChannelHeader); err != nil {
			return &Block{}, err
		}
		recode.ChannelHeader = chdr

		var shdr *common.SignatureHeader
		if shdr, err = putil.UnmarshalSignatureHeader(payload.Header.SignatureHeader); err != nil {
			return &Block{}, err
		}

		//var subject string
		var mspid *msp.SerializedIdentity
		if mspid, err = putil.UnmarshalSerializedIdentity(shdr.Creator); err != nil {
			return &Block{}, err
		}
		recode.MSPID = mspid.Mspid
		recode.Creator = string(mspid.IdBytes)
		retBlock.Data = append(retBlock.Data, recode)
	}

	return retBlock, nil
}

func ParseBlocks(blocks []*common.Block) ([]Block, error) {
	retBlocks := []Block{}
	for _, block := range blocks {
		if _bl, err := ParseBlock(block); err != nil {
			return []Block{}, err
		} else {
			retBlocks = append(retBlocks, *_bl)
		}
	}
	return retBlocks, nil
}