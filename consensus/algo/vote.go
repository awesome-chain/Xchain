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

package algo

import (
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee"
	"github.com/awesome-chain/Xchain/consensus/algo/logging"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/consensus/algo/util"
	crypto2 "github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
)

type (
	// rawVote is the inner struct which is authenticated with keys
	rawVote struct {
		_struct  struct{}       `codec:",omitempty,omitemptyarray"`
		From common.Address `codec:"from"`
		Sender   basics.Address `codec:"snd"`
		Round    basics.Round   `codec:"rnd"`
		Period   period         `codec:"per"`
		Step     step           `codec:"step"`
		Proposal proposalValue  `codec:"prop"`
	}

	// unauthenticatedVote is a vote which has not been verified
	unauthenticatedVote struct {
		_struct struct{}                            `codec:",omitempty,omitemptyarray"`
		R       rawVote                             `codec:"r"`
		Cred    committee.UnauthenticatedCredential `codec:"cred"`
		Sig     crypto.OneTimeSignature             `codec:"sig,omitempty,omitemptycheckstruct"`
		SignatureInfo crypto2.S256SignatureInfo `codec:"signature,omitempty,omitemptycheckstruct"`
	}

	// A vote is an endorsement of a particular proposal in Xchain
	vote struct {
		_struct struct{}                `codec:",omitempty,omitemptyarray"`
		R       rawVote                 `codec:"r"`
		Cred    committee.Credential    `codec:"cred"`
		Sig     crypto.OneTimeSignature `codec:"sig,omitempty,omitemptycheckstruct"`
		SignatureInfo crypto2.S256SignatureInfo `codec:"signature,omitempty,omitemptycheckstruct"`
	}

	// unauthenticatedEquivocationVote is a pair of votes which has not
	// been verified to be equivocating.
	unauthenticatedEquivocationVote struct {
		_struct   struct{}                            `codec:",omitempty,omitemptyarray"`
		Sender    basics.Address                      `codec:"snd"`
		Round     basics.Round                        `codec:"rnd"`
		Period    period                              `codec:"per"`
		Step      step                                `codec:"step"`
		Cred      committee.UnauthenticatedCredential `codec:"cred"`
		Proposals [2]proposalValue                    `codec:"props"`
		Sigs      [2]crypto.OneTimeSignature          `codec:"sigs"`
	}

	// An equivocationVote is a pair of votes from the same sender that
	// votes for two different hashes.
	//
	// These pairs are necessarily generated by a faulty node. However, if
	// we ever receive such a pair, we must count this as a single
	// "wildcard" vote to avoid violating vote propagation assumptions and
	// causing a fork.
	equivocationVote struct {
		_struct   struct{}                   `codec:",omitempty,omitemptyarray"`
		Sender    basics.Address             `codec:"snd"`
		Round     basics.Round               `codec:"rnd"`
		Period    period                     `codec:"per"`
		Step      step                       `codec:"step"`
		Cred      committee.Credential       `codec:"cred"`
		Proposals [2]proposalValue           `codec:"props"`
		Sigs      [2]crypto.OneTimeSignature `codec:"sigs"`
	}
)

// verify verifies that a vote that was received from the network is valid.
func (uv unauthenticatedVote) verify(l LedgerReader) (vote, error) {
	rv := uv.R
	m, err := membership(l, rv.Sender, rv.Round, rv.Period, rv.Step)
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not get membership parameters: %v", err)
	}

	switch rv.Step {
	case propose:
		if rv.Period == rv.Proposal.OriginalPeriod && rv.Sender != rv.Proposal.OriginalProposer {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: proposal-vote sender mismatches with proposal-value: %v != %v", rv.Sender, rv.Proposal.OriginalProposer)
		}
		// The following check could apply to all steps, but it's sufficient to only check in the propose step.
		if rv.Proposal.OriginalPeriod > rv.Period {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: proposal-vote in period %v claims to repropose block from future period %v", rv.Period, rv.Proposal.OriginalPeriod)
		}
		fallthrough
	case soft:
		fallthrough
	case cert:
		if rv.Proposal == bottom {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: votes from step %v cannot validate bottom", rv.Step)
		}
	}

	proto, err := l.ConsensusParams(ParamsRound(rv.Round))
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not get consensus params for round %d: %v", ParamsRound(rv.Round), err)
	}

	if rv.Round < m.Record.VoteFirstValid {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: vote by %v in round %d before VoteFirstValid %d: %+v", rv.Sender, rv.Round, m.Record.VoteFirstValid, uv)
	}

	if m.Record.VoteLastValid != 0 && rv.Round > m.Record.VoteLastValid {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: vote by %v in round %d after VoteLastValid %d: %+v", rv.Sender, rv.Round, m.Record.VoteLastValid, uv)
	}

	ephID := basics.OneTimeIDForRound(rv.Round, m.Record.KeyDilution(proto))
	voteID := m.Record.VoteID
	if !voteID.Verify(ephID, rv, uv.Sig) {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not verify FS signature on vote by %v given %v: %+v", rv.Sender, voteID, uv)
	}

	cred, err := uv.Cred.Verify(proto, m)
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: got a vote, but sender was not selected: %v", err)
	}

	return vote{R: rv, Cred: cred, Sig: uv.Sig}, nil
}

// verify verifies that a vote that was received from the network is valid.
func (uv unauthenticatedVote) verify2(l LedgerReader) (vote, error) {
	rv := uv.R
	m, err := membership(l, rv.Sender, rv.Round, rv.Period, rv.Step)
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not get membership parameters: %v", err)
	}

	switch rv.Step {
	case propose:
		if rv.Period == rv.Proposal.OriginalPeriod && rv.From != rv.Proposal.OriginalProposer2 {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: proposal-vote sender mismatches with proposal-value: %v != %v", rv.Sender, rv.Proposal.OriginalProposer)
		}
		// The following check could apply to all steps, but it's sufficient to only check in the propose step.
		if rv.Proposal.OriginalPeriod > rv.Period {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: proposal-vote in period %v claims to repropose block from future period %v", rv.Period, rv.Proposal.OriginalPeriod)
		}
		fallthrough
	case soft:
		fallthrough
	case cert:
		if rv.Proposal == bottom {
			return vote{}, fmt.Errorf("unauthenticatedVote.verify: votes from step %v cannot validate bottom", rv.Step)
		}
	}

	proto, err := l.ConsensusParams(ParamsRound(rv.Round))
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not get consensus params for round %d: %v", ParamsRound(rv.Round), err)
	}

	if rv.Round < m.Record.VoteFirstValid {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: vote by %v in round %d before VoteFirstValid %d: %+v", rv.Sender, rv.Round, m.Record.VoteFirstValid, uv)
	}

	if m.Record.VoteLastValid != 0 && rv.Round > m.Record.VoteLastValid {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: vote by %v in round %d after VoteLastValid %d: %+v", rv.Sender, rv.Round, m.Record.VoteLastValid, uv)
	}
	pubKey := crypto2.ToECDSAPub(uv.SignatureInfo.Pub[:])
	addr := crypto2.PubkeyToAddress(*pubKey)
	if addr != rv.From{
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not verify FS signature on vote by %v given %v: %+v", rv.Sender, nil, uv)
	}
	ok := crypto2.VerifySignature(uv.SignatureInfo.Pub[:], util.HashRep(rv), uv.SignatureInfo.Sig[:])
	if !ok{
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not verify FS signature on vote by %v given %v: %+v", rv.Sender, nil, uv)
	}
	//ephID := basics.OneTimeIDForRound(rv.Round, m.Record.KeyDilution(proto))
	//voteID := m.Record.VoteID
	//if !voteID.Verify(ephID, rv, uv.Sig) {
	//	return vote{}, fmt.Errorf("unauthenticatedVote.verify: could not verify FS signature on vote by %v given %v: %+v", rv.Sender, voteID, uv)
	//}
	cred, err := uv.Cred.Verify2(proto, m)
	if err != nil {
		return vote{}, fmt.Errorf("unauthenticatedVote.verify: got a vote, but sender was not selected: %v", err)
	}

	return vote{R: rv, Cred: cred, SignatureInfo: uv.SignatureInfo}, nil
}

// makeVote creates a new unauthenticated vote from its constituent components.
//
// makeVote returns an error it it fails.
func makeVote(rv rawVote, sk *vrf.PrivateKey, l Ledger) (unauthenticatedVote, error) {
	m, err := membership(l, rv.Sender, rv.Round, rv.Period, rv.Step)
	if err != nil {
		return unauthenticatedVote{}, fmt.Errorf("makeVote: could not get membership parameters: %v", err)
	}
	copy(m.Record.PublicKey[:], crypto2.FromECDSAPub(&sk.PrivateKey.PublicKey))

	proto, err := l.ConsensusParams(ParamsRound(rv.Round))
	if err != nil {
		return unauthenticatedVote{}, fmt.Errorf("makeVote: could not get consensus params for round %d: %v", ParamsRound(rv.Round), err)
	}

	if proto.FastPartitionRecovery {
		switch rv.Step {
		case propose, soft, cert, late, redo:
			if rv.Proposal == bottom {
				logging.Base().Panicf("makeVote: votes from step %v cannot validate bottom", rv.Step)
			}
		case down:
			if rv.Proposal != bottom {
				logging.Base().Panicf("makeVote: votes from step %v must validate bottom", rv.Step)
			}
		}
	} else {
		switch rv.Step {
		case propose, soft, cert:
			if rv.Proposal == bottom {
				logging.Base().Panicf("makeVote: votes from step %v cannot validate bottom", rv.Step)
			}
		}
	}
	hash := util.HashRep(rv)
	sig, err := crypto2.Sign(hash, sk.PrivateKey)
	if err != nil{
		return unauthenticatedVote{}, err
	}
	signatureInfo := crypto2.S256SignatureInfo{}
	copy(signatureInfo.Sig[:], sig)
	copy(signatureInfo.Pub[:], crypto2.FromECDSAPub(&sk.PrivateKey.PublicKey))
	cred := committee.MakeCredential2(sk, m.Selector)
	return unauthenticatedVote{R: rv, Cred: cred, SignatureInfo: signatureInfo}, nil
}

// ToBeHashed implements the Hashable interface.
func (rv rawVote) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.Vote, protocol.Encode(rv)
}

func (v vote) u() unauthenticatedVote {
	return unauthenticatedVote{R: v.R, Cred: v.Cred.UnauthenticatedCredential, Sig: v.Sig}
}

func (pair unauthenticatedEquivocationVote) verify(l LedgerReader) (equivocationVote, error) {
	if pair.Proposals[0] == pair.Proposals[1] {
		return equivocationVote{}, fmt.Errorf("isEquivocationPair: not an equivocation pair: identical vote (block hash %v == %v)", pair.Proposals[0], pair.Proposals[1])
	}

	rv0 := rawVote{Sender: pair.Sender, Round: pair.Round, Period: pair.Period, Step: pair.Step, Proposal: pair.Proposals[0]}
	rv1 := rawVote{Sender: pair.Sender, Round: pair.Round, Period: pair.Period, Step: pair.Step, Proposal: pair.Proposals[1]}

	uv0 := unauthenticatedVote{R: rv0, Cred: pair.Cred, Sig: pair.Sigs[0]}
	uv1 := unauthenticatedVote{R: rv1, Cred: pair.Cred, Sig: pair.Sigs[1]}

	v0, err := uv0.verify(l)
	if err != nil {
		return equivocationVote{}, fmt.Errorf("unauthenticatedEquivocationVote.verify: failed to verify pair 0: %v", err)
	}

	_, err = uv1.verify(l)
	if err != nil {
		return equivocationVote{}, fmt.Errorf("unauthenticatedEquivocationVote.verify: failed to verify pair 1: %v", err)
	}

	return equivocationVote{
		Sender:    pair.Sender,
		Round:     pair.Round,
		Period:    pair.Period,
		Step:      pair.Step,
		Cred:      v0.Cred,
		Proposals: pair.Proposals,
		Sigs:      pair.Sigs,
	}, nil
}

// the first member of the equivocation pair
func (pair equivocationVote) v0() vote {
	rv := rawVote{Sender: pair.Sender, Round: pair.Round, Period: pair.Period, Step: pair.Step, Proposal: pair.Proposals[0]}
	return vote{R: rv, Cred: pair.Cred, Sig: pair.Sigs[0]}
}

// the second member of the equivocation pair
func (pair equivocationVote) v1() vote {
	rv := rawVote{Sender: pair.Sender, Round: pair.Round, Period: pair.Period, Step: pair.Step, Proposal: pair.Proposals[1]}
	return vote{R: rv, Cred: pair.Cred, Sig: pair.Sigs[1]}
}
