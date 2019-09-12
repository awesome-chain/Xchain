package algo

import (
	"context"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto/vrf"
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
	emptyOutput = vrf.Output{0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,}//0x0000000000000000000000000000000000000000000000000000000000000000
)

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
	Validate(context.Context, types.Block) (ValidatedBlock, error)
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

// A ValidatedBlock represents an Block that has been successfuly validated
// and can now be recorded in the ledger.  This is an optimized version of
// calling EnsureBlock() on the Ledger.
type ValidatedBlock interface {
	// WithSeed creates a copy of this ValidatedBlock with its
	// cryptographically random seed set to the given value.
	//
	// Calls to Seed() or to Digest() on the copy's Block must
	// reflect the value of the new seed.
	WithSeed(common.Seed) ValidatedBlock

	// Block returns the underlying block that has been validated.
	Block() types.Block
}

