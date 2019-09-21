package agreement

import (
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee"
	"github.com/awesome-chain/Xchain/core/types"
)

type Block struct {
	*types.Block
}

//algorand block
func (b Block) WithSeed(seed committee.Seed) ValidatedBlock {
	return b
}
func (b Block) Digest() crypto.Digest {
	return crypto.Digest(b.Hash())
}

func (b Block) Round() basics.Round {
	return basics.Round(b.Number().Uint64())
}
func (b Block) Seed() committee.Seed {
	return committee.Seed(b.Header().Seed)
}

//algorand block

func (b Block) GetBlock() Block {
	return b
}
