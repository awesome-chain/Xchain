package agreement

import (
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus/algo/config"
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/core"
	"github.com/awesome-chain/Xchain/core/state"
	"github.com/awesome-chain/go-deadlock"
)

type ledger struct {
	mu    deadlock.Mutex
	chain *core.BlockChain
	db    *state.StateDB
}

func (l *ledger) NextRound() basics.Round {
	return basics.Round(l.chain.CurrentBlock().Number().Uint64() + 1)
}

// Wait returns a channel which fires when the specified round
// completes and is durably stored on disk.
func (l *ledger) Wait(basics.Round) chan struct{} {
	return nil
}

// Seed returns the VRF seed that was agreed upon in a given round.
//
// The Seed is a source of cryptographic entropy which has bounded
// bias. It is used to select committees for participation in
// sortition.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
func (l *ledger) Seed(r basics.Round) (committee.Seed, error) {
	return committee.Seed(l.chain.GetBlockByNumber(uint64(r)).Header().Seed), nil
}

// BalanceRecord returns the BalanceRecord associated with some Address
// at the conclusion of a given round.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
func (l *ledger) BalanceRecord(r basics.Round, addr basics.Address) (basics.BalanceRecord, error) {
	return basics.BalanceRecord{}, nil
}
func (l *ledger) BalanceRecord2(r basics.Round, addr common.Address) (basics.BalanceRecord, error) {
	record := basics.BalanceRecord{}
	record.Addr2 = addr
	record.AccountData.MicroAlgos = basics.MicroAlgos{
		Raw:  l.db.GetBalance(addr).Uint64(),
		Raw2: l.db.GetBalance(addr),
	}
	return record, nil
}

// Circulation returns the total amount of money in circulation at the
// conclusion of a given round.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
func (l *ledger) Circulation(basics.Round) (basics.MicroAlgos, error) {
	balance := l.db.GetBalance(common.TotalSuppyAddress)
	algos := basics.MicroAlgos{
		Raw:  balance.Uint64(),
		Raw2: balance,
	}
	return algos, nil
}

// LookupDigest returns the Digest of the entry that was agreed on in a
// given round.
//
// Recent Entry Digests are periodically used when computing the Seed.
// This prevents some subtle attacks.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
//
// A LedgerReader need only keep track of the digest from the most
// recent multiple of (config.Protocol.BalLookback/2). All other
// digests may be forgotten without hurting liveness.
func (l *ledger) LookupDigest(r basics.Round) (crypto.Digest, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if r >= l.NextRound() {
		err := fmt.Errorf("Seed called on future round: %v > %v! (this is probably a bug)", r, l.NextRound())
		panic(err)
	}
	b := l.chain.GetBlockByNumber(uint64(r))
	bb := Block{
		Block:b,
	}
	return bb.Digest(), nil
}

// ConsensusParams returns the consensus parameters that are correct
// for the given round.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
//
// TODO replace with ConsensusVersion
func (l *ledger) ConsensusParams(basics.Round) (config.ConsensusParams, error) {
	return config.Consensus[protocol.ConsensusCurrentVersion], nil
}

// ConsensusVersion returns the consensus version that is correct
// for the given round.
//
// This method returns an error if the given Round has not yet been
// confirmed. It may also return an error if the given Round is
// unavailable by the storage device. In that case, the agreement
// protocol may lose liveness.
func (l *ledger) ConsensusVersion(basics.Round) (protocol.ConsensusVersion, error) {
	return protocol.ConsensusCurrentVersion, nil
}

// EnsureBlock adds a Block, along with a Certificate authenticating
// its contents, to the ledger.
//
// The Ledger must guarantee that after this method returns, any Seed,
// Record, or Circulation call reflects the contents of this Block.
//
// EnsureBlock will never be called twice for two entries e1 and e2
// where e1.Round() == e2.Round() but e1.Digest() != e2.Digest(). If
// this is the case, the behavior of Ledger is undefined.
// (Implementations are encouraged to panic or otherwise fail loudly in
// this case, because it means that a fork has occurred.)
//
// EnsureBlock does not wait until the block is written to disk; use
// Wait() for that.
func (l *ledger) EnsureBlock(b Block, c Certificate) {
	return
}

// EnsureValidatedBlock is an optimized version of EnsureBlock that
// works on a ValidatedBlock, but otherwise has the same semantics
// as above.
func (l *ledger) EnsureValidatedBlock(e ValidatedBlock, c Certificate) {
	l.EnsureBlock(e.GetBlock(), c)
}

// EnsureDigest waits until some Block that corresponds to a given
// Certificate appears in the ledger.  EnsureDigest does not wait for
// the block to be written to disk; use Wait() if needed.
//
// The Ledger must guarantee that after this method returns, any Seed,
// Record, or Circulation call reflects the contents of the Block
// authenticated by the given Certificate.
//
// EnsureDigest will never be called twice for two certificates c1 and
// c2 where c1 authenticates the block e1 and c2 authenticates the block
// e2, but e1.Round() == e2.Round() and e1.Digest() != e2.Digest(). If
// this is the case, the behavior of Ledger is undefined.
// (Implementations are encouraged to panic or otherwise fail loudly in
// this case, because it means that a fork has occurred.)
func (l *ledger) EnsureDigest(c Certificate, cc chan struct{}, av *AsyncVoteVerifier) {

}
