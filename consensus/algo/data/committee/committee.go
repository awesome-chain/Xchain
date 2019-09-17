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
	"github.com/awesome-chain/Xchain/consensus/algo/config"
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	crypto2 "github.com/awesome-chain/Xchain/crypto"
)

// A Selector deterministically defines a cryptographic sortition committee. It
// contains both the input to the sortition VRF and the size of the sortition
// committee.
type Selector interface {
	// The hash of a struct which implements Selector is used as the input
	// to the VRF.
	crypto.Hashable

	// CommitteeSize returns the size of the committee determined by this
	// Selector.
	CommitteeSize(config.ConsensusParams) uint64
}

// Membership encodes the parameters used to verify membership in a committee.
type Membership struct {
	Record     basics.BalanceRecord
	Selector   Selector
	PublicKey  crypto2.S256PublicKey
	TotalMoney basics.MicroAlgos
}

// A Seed contains cryptographic entropy which can be used to determine a
// committee.
type Seed [32]byte

// ToBeHashed implements the crypto.Hashable interface
func (s Seed) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.Seed, s[:]
}
