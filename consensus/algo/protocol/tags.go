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

package protocol

// Tag represents a message type identifier.  Messages have a Tag field. Handlers can register to a given Tag.
// e.g., the agreement service can register to handle agreements with the Agreement tag.
type Tag string

// Tags, in lexicographic sort order to avoid duplicates.
const (
	UnknownMsgTag      Tag = "??"
	AgreementVoteTag   Tag = "AV"
	MsgSkipTag         Tag = "MS"
	NetPrioResponseTag Tag = "NP"
	PingTag            Tag = "pi"
	PingReplyTag       Tag = "pj"
	ProposalPayloadTag Tag = "PP"
	TxnTag             Tag = "TX"
	UniCatchupReqTag   Tag = "UC"
	UniCatchupResTag   Tag = "UT"
	UniEnsBlockReqTag  Tag = "UE"
	UniEnsBlockResTag  Tag = "US"
	VoteBundleTag      Tag = "VB"
)

// Complement is a convenience function for returning a corresponding response/request tag
func (t Tag) Complement() Tag {
	switch t {
	case UniCatchupResTag:
		return UniCatchupReqTag
	case UniCatchupReqTag:
		return UniCatchupResTag
	case UniEnsBlockResTag:
		return UniEnsBlockReqTag
	case UniEnsBlockReqTag:
		return UniEnsBlockResTag
	default:
		return UnknownMsgTag
	}
}
