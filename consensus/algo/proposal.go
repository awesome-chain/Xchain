package algo

import (
	"context"
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"github.com/awesome-chain/Xchain/rlp"
)

var bottom proposalValue

// A proposalValue is a triplet of a block hashes (the contents themselves and the encoding of the block),
// its proposer, and the period in which it was proposed.
type proposalValue struct {
	OriginalPeriod   period
	OriginalProposer common.Address
	BlockDigest      common.Hash
	EncodingDigest   common.Hash
}

// A unauthenticatedProposal is an Block along with everything needed to validate it.
type unauthenticatedProposal struct {
	types.Block
	SeedProof []byte
	OriginalPeriod   period
	OriginalProposer common.Address
}

// ToBeHashed implements the Hashable interface.
func (p *unauthenticatedProposal) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(p)
	if err != nil{
		return "", nil, err
	}
	return protocol.Payload, bs, nil
}

// value returns the proposal-value associated with this proposal.
func (p *unauthenticatedProposal) value() proposalValue {
	return proposalValue{
		OriginalPeriod:   p.OriginalPeriod,
		OriginalProposer: p.OriginalProposer,
		BlockDigest:      p.Hash(),
		EncodingDigest:   HashObj(p),
	}
}

// value returns the proposal-value associated with this proposal.
func (p *unauthenticatedProposal) Seed() common.Seed {
	return p.seed()
}

// value returns the proposal-value associated with this proposal.
func (p *unauthenticatedProposal) seed() common.Seed {
	return p.Header().Seed
}

// A proposal is an Block along with everything needed to validate it.
type proposal struct {
	unauthenticatedProposal

	// ve stores an optional ValidatedBlock representing this block.
	// This allows us to avoid re-computing the state delta when
	// applying this block to the ledger.  This is not serialized
	// to disk, so after a crash, we will fall back to applying the
	// raw Block to the ledger (and re-computing the state delta).
	ve ValidatedBlock
}

func makeProposal(ve ValidatedBlock, pf []byte, origPer period, origProp common.Address) proposal {
	e := ve.Block()
	var payload unauthenticatedProposal
	payload.Block = e
	payload.SeedProof = pf
	payload.OriginalPeriod = origPer
	payload.OriginalProposer = origProp
	return proposal{unauthenticatedProposal: payload, ve: ve}
}

func (p proposal) u() unauthenticatedProposal {
	return p.unauthenticatedProposal
}

// A proposerSeed is a Hashable input to proposer seed derivation.
type proposerSeed struct {
	Addr common.Address
	VRF  vrf.Output
}

// ToBeHashed implements the Hashable interface.
func (s *proposerSeed) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(s)
	if err != nil{
		return "", nil, err
	}
	return protocol.ProposerSeed, bs, nil
}

// A seedInput is a Hashable input to seed rerandomization.
type seedInput struct {
	Alpha   common.Hash
	History common.Hash
}

// ToBeHashed implements the Hashable interface.
func (i *seedInput) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(i)
	if err != nil{
		return "", nil, err
	}
	return protocol.ProposerSeed, bs, nil
}

func deriveNewSeed(address common.Address, vrfSK *vrf.PrivateKey, rnd round, period period, ledger consensus.ChainReader) (newSeed common.Seed, seedProof vrf.Proof, err error){
	var seedRound Round
	var alpha common.Hash
	var output vrf.Output
	//TODO read config
	if rnd > 2{
		seedRound = rnd - 2
	}
	prevHeader := ledger.GetHeaderByNumber(uint64(seedRound))
	prevSeed := prevHeader.Seed
	if period == 0 {
		output, seedProof = vrfSK.Evaluate(prevSeed[:])
		alpha = HashObj(&proposerSeed{
			Addr: address,
			VRF: output,
		})
	} else {
		alpha = common.Hash(prevSeed)
	}
	input := seedInput{
		Alpha: alpha,
	}
	newSeed = common.Seed(HashObj(&input))
	return
}

func verifyNewSeed(p *unauthenticatedProposal, ledger consensus.ChainReader) error {
	var seedRound Round
	var alpha common.Hash
	value := p.value()
	rnd := Round(p.Number().Uint64())
	//TODO read config
	if rnd > 2{
		seedRound = rnd - 2
	}
	prevHeader := ledger.GetHeaderByNumber(uint64(seedRound))
	prevSeed := prevHeader.Seed

	if value.OriginalPeriod == 0 {
		sig := p.Block.Header().Sig
		hash := hashHeader(p.Block.Header())
		pubKeyBS, err := crypto.Ecrecover(hash[:], sig)
		if err != nil{
			return fmt.Errorf("retrieve public key error: [%v]", err)
		}
		pubKey := crypto.ToECDSAPub(pubKeyBS)
		vrfPubkey := vrf.PublicKey{
			pubKey,
		}
		output, err := vrfPubkey.ProofToHash(prevSeed[:], p.SeedProof)
		if err != nil{
			return fmt.Errorf("verify proof error: [%v]", err)
		}
		if output == emptyOutput{
			return fmt.Errorf("verify failed: [%v]", output)
		}
		alpha = HashObj(&proposerSeed{
			Addr: p.OriginalProposer,
			VRF: output,
		})
	} else {
		alpha = common.Hash(prevSeed)
	}

	input := seedInput{Alpha: alpha}

	if p.Seed() != common.Seed(HashObj(&input)) {
		return fmt.Errorf("payload seed malformed (%v != %v)", common.Seed(HashObj(&input)), p.Seed())
	}
	return nil
}

func proposalForBlock(address common.Address, vrfSK *vrf.PrivateKey, ve ValidatedBlock, period period, ledger consensus.ChainReader) (proposal, proposalValue, error) {
	rnd := ve.Block().Number().Uint64()
	newSeed, seedProof, err := deriveNewSeed(address, vrfSK, Round(rnd), period, ledger)
	if err != nil {
		return proposal{}, proposalValue{}, fmt.Errorf("proposalForBlock: could not derive new seed: %v", err)
	}

	ve = ve.WithSeed(newSeed)
	proposal := makeProposal(ve, seedProof, period, address)
	value := proposalValue{
		OriginalPeriod:   period,
		OriginalProposer: address,
		BlockDigest:      proposal.Block.Hash(),
		EncodingDigest:   HashObj(&proposal),
	}
	return proposal, value, nil
}

// validate returns true if the proposal is valid.
// It checks the proposal seed and then calls validator.Validate.
func (p unauthenticatedProposal) validate(ctx context.Context, current round, ledger consensus.ChainReader, validator BlockValidator) (proposal, error) {
	var invalid proposal
	entry := p.Block

	if Round(entry.Number().Uint64()) != current {
		return invalid, fmt.Errorf("proposed entry from wrong round: entry.Round() != current: %v != %v", Round(entry.Number().Uint64()), current)
	}

	err := verifyNewSeed(&p, ledger)
	if err != nil {
		return invalid, fmt.Errorf("proposal has bad seed: %v", err)
	}

	ve, err := validator.Validate(ctx, entry)
	if err != nil {
		return invalid, fmt.Errorf("EntryValidator rejected entry: %v", err)
	}

	return makeProposal(ve, p.SeedProof, p.OriginalPeriod, p.OriginalProposer), nil
}