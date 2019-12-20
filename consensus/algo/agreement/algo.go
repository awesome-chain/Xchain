package agreement

import (
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus"
	"github.com/awesome-chain/Xchain/core/state"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/rpc"
	"math/big"
)

//type AlgoBlockFactory struct {
//}
//
//func (a *AlgoBlockFactory) AssembleBlock(basics.Round, time.Time) (ValidatedBlock, error) {
//	return nil, nil
//}
//
//func (a *AlgoBlockFactory) AssembleProposal(vrfSK *vrf.PrivateKey, round basics.Round, per period, leger LedgerReader) (*proposal, *proposalValue, error) {
//	address := crypto2.PubkeyToAddress(vrfSK.PrivateKey.PublicKey)
//	newSeed, seedProof, err := DeriveNewSeed2(address, vrfSK, round, per, leger)
//	if err != nil {
//		return nil, nil, err
//	}
//	header := &types.Header{
//		Coinbase: address,
//		Number:   new(big.Int).SetUint64(uint64(round)),
//	}
//
//	header.Seed = common.Seed(newSeed)
//	b := types.NewBlock(header, nil, nil, nil)
//	vb := Block{
//		Block: b,
//	}
//	var pubKey crypto2.S256PublicKey
//	copy(pubKey[:], crypto2.FromECDSAPub(&vrfSK.PrivateKey.PublicKey))
//	p := makeProposal2(vb, seedProof, per, address, pubKey)
//	pv := p.value()
//	return &p, &pv, nil
//}
type Algo struct {
}

func (*Algo) Author(header *types.Header) (common.Address, error) {
	return common.Address{}, nil
}

func (*Algo) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return nil
}

func (*Algo) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	return nil, nil
}

func (*Algo) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}

func (*Algo) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return nil
}

func (*Algo) Prepare(chain consensus.ChainReader, header *types.Header) error {
	return nil
}

func (*Algo) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	return nil, nil
}

func (*Algo) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	return nil, nil
}

func (*Algo) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return nil
}

func (*Algo) APIs(chain consensus.ChainReader) []rpc.API {
	return nil
}
