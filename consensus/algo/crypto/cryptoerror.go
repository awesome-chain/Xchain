// Copyright (C) 2019 Algorand, Inc.
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

package crypto

import "errors"

const errorinvalidversion = "Invalid version"
const errorinvalidaddress = "Invalid address"
const errorinvalidthreshold = "Invalid threshold"
const errorinvalidnumberofsignature = "Invalid number of signatures"
const errorkeynotexist = "Key does not exist"
const errorsubsigverification = "Verification failure: subsignature"
const errorkeysnotmatch = "Public key lists do not match"
const errorinvalidduplicates = "Invalid duplicates"

var errUnknownVersion = errors.New("unknown version")
