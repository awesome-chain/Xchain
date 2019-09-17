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

package committee

import (
	"encoding/binary"
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee/sortition"
	"github.com/awesome-chain/Xchain/consensus/algo/util"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"math/big"

	"github.com/awesome-chain/Xchain/consensus/algo/config"
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/logging"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	crypto2 "github.com/awesome-chain/Xchain/crypto"
	sortition2 "github.com/awesome-chain/Xchain/crypto/sortition"
)

type (
	// An UnauthenticatedCredential is a Credential which has not yet been
	// authenticated.
	UnauthenticatedCredential struct {
		_struct struct{}        `codec:",omitempty,omitemptyarray"`
		Proof   crypto.VrfProof `codec:"pf"`
		Proof2  vrf.Proof       `codec:"proof"`
	}

	// A Credential represents a proof of committee membership.
	//
	// The multiplicity of this membership is specified in the Credential's
	// weight. The VRF output hash (with the owner's address hashed in) is
	// also cached.
	//
	// Upgrades: whether or not domain separation is enabled is cached.
	// If this flag is set, this flag also includes original hashable
	// credential.
	Credential struct {
		_struct struct{}      `codec:",omitempty,omitemptyarray"`
		Weight  uint64        `codec:"wt"`
		VrfOut  crypto.Digest `codec:"h"`
		VrfOut2 vrf.Output    `codec:"output"`

		DomainSeparationEnabled bool               `codec:"ds"`
		Hashable                hashableCredential `codec:"hc"`

		UnauthenticatedCredential
	}

	hashableCredential struct {
		_struct struct{}         `codec:",omitempty,omitemptyarray"`
		RawOut  crypto.VrfOutput `codec:"v"`
		RawOut2 vrf.Output       `codec:"output"`
		Member  basics.Address   `codec:"m"`
		Member2 common.Address   `codec:"address"`
		Iter    uint64           `codec:"i"`
	}
)

// Verify an unauthenticated Credential that was received from the network.
//
// Verify checks if the given credential is a valid proof of membership
// conditioned on the provided committee membership parameters.
//
// If it is, the returned Credential constitutes a proof of this fact.
// Otherwise, an error is returned.
func (cred UnauthenticatedCredential) Verify(proto config.ConsensusParams, m Membership) (res Credential, err error) {
	selectionKey := m.Record.SelectionID
	ok, vrfOut := selectionKey.Verify(cred.Proof, m.Selector)

	hashable := hashableCredential{
		RawOut: vrfOut,
		Member: m.Record.Addr,
	}

	// Also hash in the address. This is necessary to decorrelate the selection of different accounts that have the same VRF key.
	var h crypto.Digest
	if proto.CredentialDomainSeparationEnabled {
		h = crypto.HashObj(hashable)
	} else {
		h = crypto.Hash(append(vrfOut[:], m.Record.Addr[:]...))
	}

	if !ok {
		err = fmt.Errorf("UnauthenticatedCredential.Verify: could not verify VRF Proof with %v (parameters = %+v, proof = %#v)", selectionKey, m, cred.Proof)
		return
	}

	var weight uint64
	userMoney := m.Record.VotingStake()
	expectedSelection := float64(m.Selector.CommitteeSize(proto))

	if m.TotalMoney.Raw < userMoney.Raw {
		logging.Base().Panicf("UnauthenticatedCredential.Verify: total money = %v, but user money = %v", m.TotalMoney, userMoney)
	} else if m.TotalMoney.IsZero() || expectedSelection == 0 || expectedSelection > float64(m.TotalMoney.Raw) {
		logging.Base().Panicf("UnauthenticatedCredential.Verify: m.TotalMoney %v, expectedSelection %v", m.TotalMoney.Raw, expectedSelection)
	} else if !userMoney.IsZero() {
		weight = sortition.Select(userMoney.Raw, m.TotalMoney.Raw, expectedSelection, h)
	}

	if weight == 0 {
		err = fmt.Errorf("UnauthenticatedCredential.Verify: credential has weight 0")
	} else {
		res = Credential{
			UnauthenticatedCredential: cred,
			VrfOut:                    h,
			Weight:                    weight,
			DomainSeparationEnabled:   proto.CredentialDomainSeparationEnabled,
		}
		if res.DomainSeparationEnabled {
			res.Hashable = hashable
		}
	}
	return
}

func (cred UnauthenticatedCredential) Verify2(proto config.ConsensusParams, m Membership) (res Credential, err error) {
	selectionKey := &vrf.PublicKey{
		PublicKey: crypto2.ToECDSAPub(m.Record.PublicKey[:]),
	}
	vrfOut, err := selectionKey.ProofToHash(cred.Proof2[:], util.HashRep(m.Selector))
	if vrfOut == vrf.EmptyOutput {
		err = fmt.Errorf("UnauthenticatedCredential.Verify: could not verify VRF Proof with %v (parameters = %+v, proof = %#v)", selectionKey, m, cred.Proof)
		return
	}

	hashable := hashableCredential{
		RawOut2: vrfOut,
		Member2: m.Record.Addr2,
	}

	// Also hash in the address. This is necessary to decorrelate the selection of different accounts that have the same VRF key.
	var h crypto.Digest
	if proto.CredentialDomainSeparationEnabled {
		h = crypto.HashObj(hashable)
	} else {
		h = crypto.Hash(append(vrfOut[:], m.Record.Addr2[:]...))
	}

	if err != nil {
		err = fmt.Errorf("UnauthenticatedCredential.Verify: could not verify VRF Proof with %v (parameters = %+v, proof = %#v)", selectionKey, m, cred.Proof)
		return
	}

	var weight uint64
	userMoney := m.Record.VotingStake()
	expectedSelection := float64(m.Selector.CommitteeSize(proto))

	if m.TotalMoney.Raw < userMoney.Raw {
		logging.Base().Panicf("UnauthenticatedCredential.Verify: total money = %v, but user money = %v", m.TotalMoney, userMoney)
	} else if m.TotalMoney.IsZero() || expectedSelection == 0 || expectedSelection > float64(m.TotalMoney.Raw) {
		logging.Base().Panicf("UnauthenticatedCredential.Verify: m.TotalMoney %v, expectedSelection %v", m.TotalMoney.Raw, expectedSelection)
	} else if !userMoney.IsZero() {
		weight = sortition2.Select(userMoney.Raw, m.TotalMoney.Raw, expectedSelection, h)
	}

	if weight == 0 {
		err = fmt.Errorf("UnauthenticatedCredential.Verify: credential has weight 0")
	} else {
		res = Credential{
			UnauthenticatedCredential: cred,
			VrfOut2:                   vrf.Output(h),
			Weight:                    weight,
			DomainSeparationEnabled:   proto.CredentialDomainSeparationEnabled,
		}
		if res.DomainSeparationEnabled {
			res.Hashable = hashable
		}
	}
	return
}

// MakeCredential creates a new unauthenticated Credential given some selector.
func MakeCredential(secrets *crypto.VrfPrivkey, sel Selector) UnauthenticatedCredential {
	pf, ok := secrets.Prove(sel)
	if !ok {
		logging.Base().Error("Failed to construct a VRF proof -- participation key may be corrupt")
		return UnauthenticatedCredential{}
	}
	return UnauthenticatedCredential{Proof: pf}
}

// MakeCredential creates a new unauthenticated Credential given some selector.
func MakeCredential2(sk *vrf.PrivateKey, sel Selector) UnauthenticatedCredential {
	var pf vrf.Proof
	hash := util.HashRep(sel)
	_, proof := sk.Evaluate(hash)
	copy(pf[:], proof)
	return UnauthenticatedCredential{Proof2: pf}
}

// Less returns true if this Credential is less than the other credential; false
// otherwise (i.e., >=).
//
// Precondition: both credentials have nonzero weight
func (cred Credential) Less(otherCred Credential) bool {
	i1 := cred.lowestOutput()
	i2 := otherCred.lowestOutput()

	return i1.Cmp(i2) < 0
}

// Equals compares the hash of two Credentials to determine equality and returns
// true if they're equal.
func (cred Credential) Equals(otherCred Credential) bool {
	return cred.VrfOut == otherCred.VrfOut
}

// Selected returns whether this Credential was selected (i.e., if its weight is
// greater than zero).
func (cred Credential) Selected() bool {
	return cred.Weight > 0
}

func (cred Credential) lowestOutput() *big.Int {
	var lowest big.Int

	h1 := cred.VrfOut
	for i := uint64(0); i < cred.Weight; i++ {
		var h crypto.Digest
		if cred.DomainSeparationEnabled {
			cred.Hashable.Iter = i
			h = crypto.HashObj(cred.Hashable)
		} else {
			var h2 crypto.Digest
			binary.BigEndian.PutUint64(h2[:], i)
			h = crypto.Hash(append(h1[:], h2[:]...))
		}

		if i == 0 {
			lowest.SetBytes(h[:])
		} else {
			var temp big.Int
			temp.SetBytes(h[:])
			if temp.Cmp(&lowest) < 0 {
				lowest.Set(&temp)
			}
		}
	}

	return &lowest
}

func (cred hashableCredential) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.Credential, protocol.Encode(cred)
}