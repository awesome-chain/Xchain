package algo

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/awesome-chain/Xchain/consensus"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"math/big"
	"sync"
	"time"
)

// AssemblyTime is the max amount of time to spend on generating a proposal block.
const AssemblyTime time.Duration = 250 * time.Millisecond

// TODO put these in config
const (
	pseudonodeVerificationBacklog = 32
)
var errPseudonodeBacklogFull = fmt.Errorf("pseudonode input channel is full")
var errPseudonodeVerifierClosedChannel = fmt.Errorf("crypto verifier closed the output channel prematurely")
var errPseudonodeNoVotes = fmt.Errorf("no valid participation keys to generate votes for given round")
var errPseudonodeNoProposals = fmt.Errorf("no valid participation keys to generate proposals for given round")


type AsyncPseudoNode struct {
	factory   BlockFactory
	validator BlockValidator
	ledger            consensus.ChainReader
	quit              chan struct{}
	closeWG           *sync.WaitGroup
	proposalsVerifier *PseudoNodeVerifier
}

// PseudoNodeTask encapsulates a single task which should be executed by the pseudonode.
type PseudoNodeTask interface {
	// Execute a task with a given cryptoVerifier and quit channel.
	execute(verifier *AsyncVoteVerifier, quit chan struct{})
}

type PseudoNodeBaseTask struct {
	node    *AsyncPseudoNode
	context context.Context // the context associated with that task; context might expire for a single task but remain valid for others.
	out     chan externalEvent
}

type PseudoNodeProposalsTask struct {
	PseudoNodeBaseTask
	round  uint64
	period uint64
}

type PseudoNodeVerifier struct {
	verifier      *AsyncVoteVerifier
	incomingTasks chan PseudoNodeTask
}

func (n *AsyncPseudoNode) MakeProposals(ctx context.Context, r uint64, p uint64) (<-chan externalEvent, error) {
	proposalTask := n.makeProposalsTask(ctx, r, p)
	select {
	case n.proposalsVerifier.incomingTasks <- proposalTask:
		return proposalTask.outputChannel(), nil
	default:
		proposalTask.close()
		return nil, errPseudonodeBacklogFull
	}
}

// makeProposals creates a slice of block proposals for the given round and period.
func (n *AsyncPseudoNode) makeProposals(round uint64, period uint64, sk *ecdsa.PrivateKey) []*Proposal {
	//deadline := time.Now().Add(AssemblyTime)
	proposals := make([]*Proposal, 0)
	address := crypto.PubkeyToAddress(sk.PublicKey)
	newSeed, seedProof, err := DeriveNewSeed(address, &vrf.PrivateKey{sk}, round, period, n.ledger)
	if err != nil {
		return nil
	}
	header := &types.Header{
		Coinbase: address,
		Number:   new(big.Int).SetUint64(round),
	}
	header.Seed = newSeed
	b := types.NewBlock(header, nil, nil, nil)
	p := MakeProposal(b, seedProof[:], 0, &sk.PublicKey)
	// create the block proposal
	proposals = append(proposals, p)
	return proposals
}


func (n *AsyncPseudoNode) makeProposalsTask(ctx context.Context, r uint64, p uint64) *PseudoNodeProposalsTask {
	pt := &PseudoNodeProposalsTask{
		PseudoNodeBaseTask: PseudoNodeBaseTask{
			node:    n,
			context: ctx,
			out:     make(chan externalEvent),
		},
		round:  r,
		period: p,
	}
	return pt
}

func (t *PseudoNodeProposalsTask) execute(verifier *AsyncVoteVerifier, quit chan struct{}) {
	defer t.close()
	// check to see if task already expired.
	//if t.context.Err() != nil {
	//	return
	//}
	//payloads, votes := t.node.makeProposals(t.round, t.period, t.participation)
	return
}

func (t *PseudoNodeBaseTask) outputChannel() chan externalEvent {
	return t.out
}

func (t *PseudoNodeBaseTask) close() {
	close(t.out)
}

//func (n *AsyncPseudoNode) Quit() {
//	// protect against double-quits.
//	select {
//	case <-n.quit:
//		// if we already quit, just exit.
//		return
//	default:
//	}
//	close(n.quit)
//	n.proposalsVerifier.close()
//	n.votesVerifier.close()
//	n.closeWg.Wait()
//}

// PseudoNodeTask encapsulates a single task which should be executed by the pseudonode.
//type PseudoNodeTask interface {
//	// Execute a task with a given cryptoVerifier and quit channel.
//	execute(verifier *AsyncVoteVerifier, quit chan struct{})
//}
