package algo

import (
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"time"
)

type Algo struct {
}

func (a *Algo) AssembleBlock(basics.Round, time.Time) (ValidatedBlock, error) {
	return nil, nil
}

func (a *Algo) AssembleProposal(basics.Round, period) (*proposal, *proposalValue, error) {
	return nil, nil, nil
}
