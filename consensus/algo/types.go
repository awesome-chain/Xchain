package algo

import (
	"context"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"time"
)

// Round represents a protocol round index
type Round uint64

type (
	// round denotes a single round of the agreement protocol
	round = Round

	// step is a sequence number denoting distinct stages in Algorand
	step uint64

	// period is used to track progress with a given round in the protocol
	period uint64
)

// Algorand 2.0 steps
const (
	propose step = iota
	soft
	cert
	next
)
const (
	late step = 253 + iota
	redo
	down
)

var (
	emptyOutput = vrf.Output{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} //0x0000000000000000000000000000000000000000000000000000000000000000
)

type BlockFactory interface {
	// AssembleBlock produces a new ValidatedBlock which is suitable for proposal
	// at a given Round.  The time argument specifies a target deadline by
	// which the block should be produced.  Specifically, the deadline can
	// cause the factory to add fewer transactions to the block in question
	// than might otherwise be possible.
	//
	// AssembleBlock should produce a ValidatedBlock for which the corresponding
	// BlockValidator validates (i.e. for which BlockValidator.Validate
	// returns true). If an insufficient number of nodes can assemble valid
	// entries, the agreement protocol may lose liveness.
	//
	// AssembleBlock may return an error if the BlockFactory is unable to
	// produce a ValidatedBlock for the given round. If an insufficient number of
	// nodes on the network can assemble entries, the agreement protocol may
	// lose liveness.
	AssembleBlock(uint64, time.Time) (*types.Block, error)
}


type BlockValidator interface {
	// Validate must return an error if a given Block cannot be determined
	// to be valid as applied to the agreement state; otherwise, it returns
	// nil.
	//
	// The correctness of Validate is essential to the correctness of the
	// protocol. If Validate accepts an invalid Block (i.e., a false
	// positive), the agreement protocol may fork, or the system state may
	// even become undefined. If Validate rejects a valid Block (i.e., a
	// false negative), the agreement protocol may even lose
	// liveness. Validate should therefore be conservative in which Entries
	// it accepts.
	//
	// TODO There should probably be a second Round argument here.
	Validate(context.Context, *types.Block) (*types.Block, error)
}

// A message represents an internal message which is passed between components
// of the agreement service.
type Message struct {
	Proposal *Proposal
	UnauthenticatedProposal *UnauthenticatedProposal
}

// Hashable is an interface implemented by an object that can be represented
// with a sequence of bytes to be hashed or signed, together with a type ID
// to distinguish different types of objects.
type Hashable interface {
	ToBeHashed() (protocol.HashID, []byte, error)
}

func hashRep(h Hashable) []byte {
	hashid, data, _ := h.ToBeHashed()
	return append([]byte(hashid), data...)
}
