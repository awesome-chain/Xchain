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

package config

import (
	"testing"
)

func TestConsensusParams(t *testing.T) {
	for proto, params := range Consensus {
		// Our implementation of Payset.Commit() assumes that
		// SupportSignedTxnInBlock implies PaysetCommitFlat.
		if params.SupportSignedTxnInBlock && !params.PaysetCommitFlat {
			t.Errorf("Protocol %s: SupportSignedTxnInBlock without PaysetCommitFlat", proto)
		}

		// ApplyData requires PaysetCommitFlat.
		if params.ApplyData && !params.PaysetCommitFlat {
			t.Errorf("Protocol %s: ApplyData without PaysetCommitFlat", proto)
		}
	}
}