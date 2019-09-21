// Copyright (C) 2019 Xchain, Inc.
// This file is part of Xchain
//
// Xchain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// Xchain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with Xchain.  If not, see <https://www.gnu.org/licenses/>.

package agreement

import (
	"context"
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/crypto/vrf"

	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee"
	"github.com/awesome-chain/Xchain/consensus/algo/logging"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	crypto2 "github.com/awesome-chain/Xchain/crypto"
)

var bottom proposalValue

// A proposalValue is a triplet of a block hashes (the contents themselves and the encoding of the block),
// its proposer, and the period in which it was proposed.
type proposalValue struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	OriginalPeriod    period         `codec:"oper"`
	OriginalProposer  basics.Address `codec:"oprop"`
	OriginalProposer2 common.Address `codec:"oriprop"`
	BlockDigest       crypto.Digest  `codec:"dig"`    // = proposal.Block.Digest()
	EncodingDigest    crypto.Digest  `codec:"encdig"` // = crypto.HashObj(proposal)
}

// A transmittedPayload is the representation of a proposal payload on the wire.
type transmittedPayload struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	unauthenticatedProposal
	PriorVote unauthenticatedVote `codec:"pv"`
}

// A unauthenticatedProposal is an Block along with everything needed to validate it.
type unauthenticatedProposal struct {
	_struct struct{} `codec:",omitempty,omitemptyarray"`

	Block
	SeedProof  crypto.VrfProof `codec:"sdpf"`
	SeedProof2 vrf.Proof       `codec:"seedpf"`

	OriginalPeriod    period                `codec:"oper"`
	OriginalProposer  basics.Address        `codec:"oprop"`
	OriginalProposer2 common.Address        `codec:"oriprop"`
	OriginalPubKey    crypto2.S256PublicKey `codec:"oproppubkey"`
}

// ToBeHashed implements the Hashable interface.
func (p unauthenticatedProposal) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.Payload, protocol.Encode(p)
}

// value returns the proposal-value associated with this proposal.
func (p unauthenticatedProposal) value() proposalValue {
	return proposalValue{
		OriginalPeriod:    p.OriginalPeriod,
		OriginalProposer:  p.OriginalProposer,
		OriginalProposer2: p.OriginalProposer2,
		BlockDigest:       p.Digest(),
		EncodingDigest:    crypto.HashObj(p),
	}
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

func makeProposal(ve ValidatedBlock, pf crypto.VrfProof, origPer period, origProp basics.Address) proposal {
	e := ve.GetBlock()
	var payload unauthenticatedProposal
	payload.Block = e
	payload.SeedProof = pf
	payload.OriginalPeriod = origPer
	payload.OriginalProposer = origProp
	return proposal{unauthenticatedProposal: payload, ve: ve}
}

func makeProposal2(ve ValidatedBlock, pf vrf.Proof, origPer period, origProp common.Address, origPubKey crypto2.S256PublicKey) proposal {
	e := ve.GetBlock()
	var payload unauthenticatedProposal
	payload.Block = e
	payload.SeedProof2 = pf
	payload.OriginalPeriod = origPer
	payload.OriginalProposer2 = origProp
	payload.OriginalPubKey = origPubKey
	return proposal{unauthenticatedProposal: payload, ve: ve}
}

func (p proposal) u() unauthenticatedProposal {
	return p.unauthenticatedProposal
}

// A proposerSeed is a Hashable input to proposer seed derivation.
type proposerSeed struct {
	Addr  basics.Address   `codec:"addr"`
	Addr2 common.Address   `codec:"address"`
	VRF   crypto.VrfOutput `codec:"vrf"`
	VRF2  vrf.Output       `codec:"output"`
}

// ToBeHashed implements the Hashable interface.
func (s proposerSeed) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.ProposerSeed, protocol.Encode(s)
}

// A seedInput is a Hashable input to seed rerandomization.
type seedInput struct {
	Alpha   crypto.Digest `codec:"alpha"`
	History crypto.Digest `codec:"hist"`
}

// ToBeHashed implements the Hashable interface.
func (i seedInput) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.ProposerSeed, protocol.Encode(i)
}

func deriveNewSeed(address basics.Address, vrf *crypto.VRFSecrets, rnd round, period period, ledger LedgerReader) (newSeed committee.Seed, seedProof crypto.VRFProof, reterr error) {
	var ok bool
	var vrfOut crypto.VrfOutput

	cparams, err := ledger.ConsensusParams(ParamsRound(rnd))
	if err != nil {
		err = fmt.Errorf("failed to obtain consensus parameters in round %v: %v", ParamsRound(rnd), err)
		return
	}
	var alpha crypto.Digest
	prevSeed, err := ledger.Seed(seedRound(rnd, cparams))
	if err != nil {
		reterr = fmt.Errorf("failed read seed of round %v: %v", seedRound(rnd, cparams), err)
		return
	}

	if period == 0 {
		seedProof, ok = vrf.SK.Prove(prevSeed)
		if !ok {
			reterr = fmt.Errorf("could not make seed proof")
			return
		}
		vrfOut, ok = seedProof.Hash()
		if !ok {
			// If proof2hash fails on a proof we produced with VRF Prove, this indicates our VRF code has a dangerous bug.
			// Panicking is the only safe thing to do.
			logging.Base().Panicf("VrfProof.Hash() failed on a proof we ourselves generated; this indicates a bug in the VRF code: %v", seedProof)
		}
		alpha = crypto.HashObj(proposerSeed{Addr: address, VRF: vrfOut})
	} else {
		alpha = crypto.HashObj(prevSeed)
	}

	input := seedInput{Alpha: alpha}
	rerand := rnd % basics.Round(cparams.SeedLookback*cparams.SeedRefreshInterval)
	if rerand < basics.Round(cparams.SeedLookback) {
		digrnd := rnd.SubSaturate(basics.Round(cparams.SeedLookback * cparams.SeedRefreshInterval))
		oldDigest, err := ledger.LookupDigest(digrnd)
		if err != nil {
			reterr = fmt.Errorf("could not lookup old entry digest (for seed) from round %v: %v", digrnd, err)
			return
		}
		input.History = oldDigest
	}
	newSeed = committee.Seed(crypto.HashObj(input))
	return
}

func verifyNewSeed(p unauthenticatedProposal, ledger LedgerReader) error {
	value := p.value()
	rnd := p.Round()
	cparams, err := ledger.ConsensusParams(ParamsRound(rnd))
	if err != nil {
		return fmt.Errorf("failed to obtain consensus parameters in round %v: %v", ParamsRound(rnd), err)
	}

	balanceRound := balanceRound(rnd, cparams)
	proposerRecord, err := ledger.BalanceRecord(balanceRound, value.OriginalProposer)
	if err != nil {
		return fmt.Errorf("failed to obtain balance record for address %v in round %v: %v", value.OriginalProposer, balanceRound, err)
	}

	var alpha crypto.Digest
	prevSeed, err := ledger.Seed(seedRound(rnd, cparams))
	if err != nil {
		return fmt.Errorf("failed read seed of round %v: %v", seedRound(rnd, cparams), err)
	}

	if value.OriginalPeriod == 0 {
		verifier := proposerRecord.SelectionID
		ok, vrfOut := verifier.Verify(p.SeedProof, prevSeed)
		if !ok {
			return fmt.Errorf("payload seed proof malformed (%v, %v)", prevSeed, p.SeedProof)
		}
		vrfOut, ok = p.SeedProof.Hash()
		if !ok {
			// If proof2hash fails on a proof we produced with VRF Prove, this indicates our VRF code has a dangerous bug.
			// Panicking is the only safe thing to do.
			logging.Base().Panicf("VrfProof.Hash() failed on a proof we ourselves generated; this indicates a bug in the VRF code: %v", p.SeedProof)
		}
		alpha = crypto.HashObj(proposerSeed{Addr: proposerRecord.Addr, VRF: vrfOut})
	} else {
		alpha = crypto.HashObj(prevSeed)
	}

	input := seedInput{Alpha: alpha}
	rerand := rnd % basics.Round(cparams.SeedLookback*cparams.SeedRefreshInterval)
	if rerand < basics.Round(cparams.SeedLookback) {
		digrnd := rnd.SubSaturate(basics.Round(cparams.SeedLookback * cparams.SeedRefreshInterval))
		oldDigest, err := ledger.LookupDigest(digrnd)
		if err != nil {
			return fmt.Errorf("could not lookup old entry digest (for seed) from round %v: %v", digrnd, err)
		}
		input.History = oldDigest
	}
	if p.Seed() != committee.Seed(crypto.HashObj(input)) {
		return fmt.Errorf("payload seed malformed (%v != %v)", committee.Seed(crypto.HashObj(input)), p.Seed())
	}
	return nil
}

func proposalForBlock(address basics.Address, vrf *crypto.VRFSecrets, ve ValidatedBlock, period period, ledger LedgerReader) (proposal, proposalValue, error) {
	rnd := ve.GetBlock().Round()
	newSeed, seedProof, err := deriveNewSeed(address, vrf, rnd, period, ledger)
	if err != nil {
		return proposal{}, proposalValue{}, fmt.Errorf("proposalForBlock: could not derive new seed: %v", err)
	}

	ve = ve.WithSeed(newSeed)
	proposal := makeProposal(ve, seedProof, period, address)
	value := proposalValue{
		OriginalPeriod:   period,
		OriginalProposer: address,
		BlockDigest:      proposal.Block.Digest(),
		EncodingDigest:   crypto.HashObj(proposal),
	}
	return proposal, value, nil
}

// validate returns true if the proposal is valid.
// It checks the proposal seed and then calls validator.Validate.
func (p unauthenticatedProposal) validate(ctx context.Context, current round, ledger LedgerReader, validator BlockValidator) (proposal, error) {
	var invalid proposal
	entry := p.Block

	if entry.Round() != current {
		return invalid, fmt.Errorf("proposed entry from wrong round: entry.Round() != current: %v != %v", entry.Round(), current)
	}

	err := verifyNewSeed(p, ledger)
	if err != nil {
		return invalid, fmt.Errorf("proposal has bad seed: %v", err)
	}

	ve, err := validator.Validate(ctx, entry)
	if err != nil {
		return invalid, fmt.Errorf("EntryValidator rejected entry: %v", err)
	}

	return makeProposal(ve, p.SeedProof, p.OriginalPeriod, p.OriginalProposer), nil
}

func DeriveNewSeed2(address common.Address, vrfSK *vrf.PrivateKey, rnd round, period period, ledger LedgerReader) (newSeed committee.Seed, seedProof vrf.Proof, err error) {
	var alpha crypto.Digest
	cparams, err := ledger.ConsensusParams(ParamsRound(rnd))
	if err != nil {
		err = fmt.Errorf("failed to obtain consensus parameters in round %v: %v", ParamsRound(rnd), err)
		return
	}
	prevSeed, err := ledger.Seed(seedRound(rnd, cparams))
	if err != nil {
		return
	}
	if period == 0 {
		output, proof := vrfSK.Evaluate(prevSeed[:])
		alpha = crypto.HashObj(proposerSeed{
			Addr2: address,
			VRF2:  output,
		})
		copy(seedProof[:], proof)
	} else {
		alpha = crypto.HashObj(prevSeed)
	}

	input := seedInput{Alpha: alpha}
	rerand := rnd % basics.Round(cparams.SeedLookback*cparams.SeedRefreshInterval)
	if rerand < basics.Round(cparams.SeedLookback) {
		digrnd := rnd.SubSaturate(basics.Round(cparams.SeedLookback * cparams.SeedRefreshInterval))
		oldDigest, err0 := ledger.LookupDigest(digrnd)
		if err0 != nil {
			err = fmt.Errorf("could not lookup old entry digest (for seed) from round %v: %v", digrnd, err0)
			return
		}
		input.History = oldDigest
	}
	newSeed = committee.Seed(crypto.HashObj(input))
	return
}

func verifyNewSeed2(p unauthenticatedProposal, ledger LedgerReader) error {
	value := p.value()
	rnd := p.Round()
	cparams, err := ledger.ConsensusParams(ParamsRound(rnd))
	if err != nil {
		return fmt.Errorf("failed to obtain consensus parameters in round %v: %v", ParamsRound(rnd), err)
	}
	var alpha crypto.Digest
	prevSeed, err := ledger.Seed(seedRound(rnd, cparams))
	if err != nil {
		return fmt.Errorf("failed read seed of round %v: %v", seedRound(rnd, cparams), err)
	}

	if value.OriginalPeriod == 0 {
		pubKey := vrf.PublicKey{
			PublicKey: crypto2.ToECDSAPub(p.OriginalPubKey[:]),
		}
		output, err := pubKey.ProofToHash(prevSeed[:], p.SeedProof2[:])
		if output == vrf.EmptyOutput || err != nil {
			return fmt.Errorf("payload seed proof malformed (%v, %v)", prevSeed, p.SeedProof)
		}
		alpha = crypto.HashObj(proposerSeed{Addr2: value.OriginalProposer2, VRF2: vrf.Output(output)})
	} else {
		alpha = crypto.HashObj(prevSeed)
	}

	input := seedInput{Alpha: alpha}
	rerand := rnd % basics.Round(cparams.SeedLookback*cparams.SeedRefreshInterval)
	if rerand < basics.Round(cparams.SeedLookback) {
		digrnd := rnd.SubSaturate(basics.Round(cparams.SeedLookback * cparams.SeedRefreshInterval))
		oldDigest, err := ledger.LookupDigest(digrnd)
		if err != nil {
			return fmt.Errorf("could not lookup old entry digest (for seed) from round %v: %v", digrnd, err)
		}
		input.History = oldDigest
	}
	if p.Seed() != committee.Seed(crypto.HashObj(input)) {
		return fmt.Errorf("payload seed malformed (%v != %v)", committee.Seed(crypto.HashObj(input)), p.Seed())
	}
	return nil
}

//
//// Validate returns true if the proposal is valid.
//// It checks the proposal seed and then calls validator.Validate.
func (p unauthenticatedProposal) validate2(ctx context.Context, current basics.Round, ledger LedgerReader, validator BlockValidator) (proposal, error) {
	var invalid proposal
	entry := p.Block

	if entry.Round() != current {
		return invalid, fmt.Errorf("proposed entry from wrong round: entry.Round() != current: %v != %v", entry.Round(), current)
	}

	err := verifyNewSeed(p, ledger)
	if err != nil {
		return invalid, fmt.Errorf("proposal has bad seed: %v", err)
	}

	ve, err := validator.Validate(ctx, entry)
	if err != nil {
		return invalid, fmt.Errorf("EntryValidator rejected entry: %v", err)
	}

	return makeProposal2(ve, p.SeedProof2, p.OriginalPeriod, p.OriginalProposer2, p.OriginalPubKey), nil
}