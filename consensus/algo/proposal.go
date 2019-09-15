package algo

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"github.com/awesome-chain/Xchain/rlp"
)

//var bottom ProposalValue

// ProposalValue is a triplet of a block hashes (the contents themselves and the encoding of the block),
// its proposer, and the period in which it was proposed.
type ProposalValue struct {
	OriginalPeriod   uint64
	OriginalProposer []byte
	BlockDigest      common.Hash
	EncodingDigest   common.Hash
}

// UnauthenticatedProposal is an Block along with everything needed to validate it.
type UnauthenticatedProposal struct {
	*types.Block
	SeedProof        []byte
	OriginalPeriod   uint64
	OriginalProposer []byte
}

// ToBeHashed implements the Hashable interface.
func (p *UnauthenticatedProposal) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(p)
	if err != nil {
		return "", nil, err
	}
	return protocol.Payload, bs, nil
}

// value returns the proposal-value associated with this proposal.
func (p *UnauthenticatedProposal) Value() *ProposalValue {
	return &ProposalValue{
		OriginalPeriod:   p.OriginalPeriod,
		OriginalProposer: p.OriginalProposer,
		BlockDigest:      p.Hash(),
		EncodingDigest:   HashObj(p),
	}
}

// value returns the proposal-value associated with this proposal.
func (p *UnauthenticatedProposal) Seed() common.Seed {
	return p.seed()
}

// value returns the proposal-value associated with this proposal.
func (p *UnauthenticatedProposal) seed() common.Seed {
	return p.Header().Seed
}

// A proposal is an Block along with everything needed to validate it.
type Proposal struct {
	UnauthenticatedProposal

	// ve stores an optional ValidatedBlock representing this block.
	// This allows us to avoid re-computing the state delta when
	// applying this block to the ledger.  This is not serialized
	// to disk, so after a crash, we will fall back to applying the
	// raw Block to the ledger (and re-computing the state delta).
	ve *types.Block
}

func MakeProposal(b *types.Block, pf []byte, origPer uint64, origProp *ecdsa.PublicKey) *Proposal {
	p := &Proposal{}
	p.Block = b
	p.SeedProof = pf
	p.OriginalPeriod = origPer
	p.OriginalProposer = crypto.FromECDSAPub(origProp)
	p.ve = b
	return p
}

func (p *Proposal) U() *UnauthenticatedProposal {
	return &p.UnauthenticatedProposal
}

// A proposerSeed is a Hashable input to proposer seed derivation.
type ProposerSeed struct {
	Addr common.Address
	VRF  vrf.Output
}

// ToBeHashed implements the Hashable interface.
func (s *ProposerSeed) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(s)
	if err != nil {
		return "", nil, err
	}
	return protocol.ProposerSeed, bs, nil
}

// A seedInput is a Hashable input to seed rerandomization.
type SeedInput struct {
	Alpha   common.Hash
	History common.Hash
}

// ToBeHashed implements the Hashable interface.
func (i *SeedInput) ToBeHashed() (protocol.HashID, []byte, error) {
	bs, err := rlp.EncodeToBytes(i)
	if err != nil {
		return "", nil, err
	}
	return protocol.ProposerSeed, bs, nil
}

func DeriveNewSeed(address common.Address, vrfSK *vrf.PrivateKey, rnd uint64, period uint64, ledger consensus.ChainReader) (newSeed common.Seed, seedProof vrf.Proof, err error) {
	var seedRound uint64
	var alpha common.Hash
	var output vrf.Output
	//TODO read config
	if rnd > 2 {
		seedRound = rnd - 2
	}
	prevHeader := ledger.GetHeaderByNumber(uint64(seedRound))
	prevSeed := prevHeader.Seed
	if period == 0 {
		output, seedProof = vrfSK.Evaluate(prevSeed[:])
		alpha = HashObj(&ProposerSeed{
			Addr: address,
			VRF:  output,
		})
	} else {
		alpha = common.Hash(prevSeed)
	}
	input := SeedInput{
		Alpha: alpha,
	}
	newSeed = common.Seed(HashObj(&input))
	return
}

func VerifyNewSeed(p *UnauthenticatedProposal, ledger consensus.ChainReader) error {
	var seedRound uint64
	var alpha common.Hash
	value := p.Value()
	rnd := p.Number().Uint64()
	//TODO read config
	if rnd > 2 {
		seedRound = rnd - 2
	}
	prevHeader := ledger.GetHeaderByNumber(uint64(seedRound))
	prevSeed := prevHeader.Seed

	if value.OriginalPeriod == 0 {
		pubKey := crypto.ToECDSAPub(p.OriginalProposer)
		var vrfPubKey vrf.PublicKey
		vrfPubKey.PublicKey = pubKey
		output, err := vrfPubKey.ProofToHash(prevSeed[:], p.SeedProof[:])
		if err != nil {
			return fmt.Errorf("verify proof error: [%v]", err)
		}
		if output == emptyOutput {
			return fmt.Errorf("verify failed: [%v]", output)
		}
		alpha = HashObj(&ProposerSeed{
			Addr: crypto.PubkeyToAddress(*pubKey),
			VRF:  output,
		})
	} else {
		alpha = common.Hash(prevSeed)
	}

	input := SeedInput{Alpha: alpha}

	if p.Seed() != common.Seed(HashObj(&input)) {
		return fmt.Errorf("payload seed malformed (%v != %v)", common.Seed(HashObj(&input)), p.Seed())
	}
	return nil
}

// Validate returns true if the proposal is valid.
// It checks the proposal seed and then calls validator.Validate.
func (p UnauthenticatedProposal) Validate(ctx context.Context, current uint64, ledger consensus.ChainReader, validator BlockValidator) (*Proposal, error) {
	entry := p.Block

	if entry.Number().Uint64() != current {
		return nil, fmt.Errorf("proposed entry from wrong round: entry.Round() != current: %v != %v", Round(entry.Number().Uint64()), current)
	}

	err := VerifyNewSeed(&p, ledger)
	if err != nil {
		return nil, fmt.Errorf("proposal has bad seed: %v", err)
	}

	ve, err := validator.Validate(ctx, entry)
	if err != nil {
		return nil, fmt.Errorf("EntryValidator rejected entry: %v", err)
	}
	pubKey := crypto.ToECDSAPub(p.OriginalProposer)
	return MakeProposal(ve, p.SeedProof, p.OriginalPeriod, pubKey), nil
}
