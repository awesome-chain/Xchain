// Copyright (C) 2019 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

// Package gossip adapts the interface of network.GossipNode to
// agreement.Network.
package agreement

import (
	"context"
	"github.com/awesome-chain/Xchain/consensus/algo/logging"
	"github.com/awesome-chain/Xchain/consensus/algo/network"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/consensus/algo/util/metrics"
	"time"
)

var (
	voteBufferSize     = 10000
	proposalBufferSize = 14
	bundleBufferSize   = 7
)

var messagesHandled = metrics.MakeCounter(metrics.AgreementMessagesHandled)
var messagesDropped = metrics.MakeCounter(metrics.AgreementMessagesDropped)

type messageMetadata struct {
	raw network.IncomingMessage
}

// networkImpl wraps network.GossipNode to provide a compatible interface with agreement.
type networkImpl struct {
	voteCh     chan Message
	proposalCh chan Message
	bundleCh   chan Message

	net network.GossipNode
}

// WrapNetwork adapts a network.GossipNode into an agreement.Network.
func WrapNetwork(net network.GossipNode) Network {
	i := new(networkImpl)

	i.voteCh = make(chan Message, voteBufferSize)
	i.proposalCh = make(chan Message, proposalBufferSize)
	i.bundleCh = make(chan Message, bundleBufferSize)

	i.net = net

	handlers := []network.TaggedMessageHandler{
		{Tag: protocol.AgreementVoteTag, MessageHandler: network.HandlerFunc(i.processVoteMessage)},
		{Tag: protocol.ProposalPayloadTag, MessageHandler: network.HandlerFunc(i.processProposalMessage)},
		{Tag: protocol.VoteBundleTag, MessageHandler: network.HandlerFunc(i.processBundleMessage)},
	}
	net.RegisterHandlers(handlers)
	return i
}

func messageMetadataFromHandle(h MessageHandle) *messageMetadata {
	if msg, isMsg := h.(*messageMetadata); isMsg {
		return msg
	}
	return nil
}

func (i *networkImpl) processVoteMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.voteCh)
}

func (i *networkImpl) processProposalMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.proposalCh)
}

func (i *networkImpl) processBundleMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.bundleCh)
}

// i.e. process<Type>Message
func (i *networkImpl) processMessage(raw network.IncomingMessage, submit chan<- Message) network.OutgoingMessage {
	metadata := &messageMetadata{raw: raw}

	select {
	case submit <- Message{MessageHandle: MessageHandle(metadata), Data: raw.Data}:
		// It would be slightly better to measure at de-queue
		// time, but that happens in many places in code and
		// this is much easier.
		messagesHandled.Inc(nil)
	default:
		messagesDropped.Inc(nil)
	}

	// Immediately ignore everything here, sometimes Relay/Broadcast/Disconnect later based on API handles saved from IncomingMessage
	return network.OutgoingMessage{Action: network.Ignore}
}

func (i *networkImpl) Messages(t protocol.Tag) <-chan Message {
	switch t {
	case protocol.AgreementVoteTag:
		return i.voteCh
	case protocol.ProposalPayloadTag:
		return i.proposalCh
	case protocol.VoteBundleTag:
		return i.bundleCh
	default:
		logging.Base().Panicf("bad tag! %v", t)
		return nil
	}
}

func (i *networkImpl) Broadcast(t protocol.Tag, data []byte) (err error) {
	err = i.net.Broadcast(context.Background(), t, data, false, nil)
	if err != nil {
		logging.Base().Infof("agreement: could not broadcast message with tag %v: %v", t, err)
	}
	return
}

func (i *networkImpl) Relay(h MessageHandle, t protocol.Tag, data []byte) (err error) {
	metadata := messageMetadataFromHandle(h)
	if metadata == nil { // synthentic loopback
		err = i.net.Broadcast(context.Background(), t, data, false, nil)
		if err != nil {
			logging.Base().Infof("agreement: could not (pseudo)relay message with tag %v: %v", t, err)
		}
	} else {
		err = i.net.Relay(context.Background(), t, data, false, metadata.raw.Sender)
		if err != nil {
			logging.Base().Infof("agreement: could not relay message from %v with tag %v: %v", metadata.raw.Sender, t, err)
		}
	}
	return
}

func (i *networkImpl) Disconnect(h MessageHandle) {
	metadata := messageMetadataFromHandle(h)

	if metadata == nil { // synthentic loopback
		// TODO warn
		return
	}

	i.net.Disconnect(metadata.raw.Sender)
}

// broadcastTimeout is currently only used by test code.
// In test code we want to queue up a bunch of outbound packets and then see that they got through, so we need to wait at least a little bit for them to all go out.
// Normal agreement state machine code uses GossipNode.Broadcast non-blocking and may drop outbound packets.
func (i *networkImpl) broadcastTimeout(t protocol.Tag, data []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return i.net.Broadcast(ctx, t, data, true, nil)
}

// networkImpl wraps network.GossipNode to provide a compatible interface with agreement.
type networkImplSimulate struct {
	voteCh     chan Message
	proposalCh chan Message
	bundleCh   chan Message

	net network.GossipNode
}

// WrapNetwork adapts a network.GossipNode into an agreement.Network.
func WrapNetworkSimulate(net network.GossipNode) Network {
	//i := new(networkImplSimulate)
	i := new(networkImpl)

	i.voteCh = make(chan Message, voteBufferSize)
	i.proposalCh = make(chan Message, proposalBufferSize)
	i.bundleCh = make(chan Message, bundleBufferSize)

	i.net = net

	handlers := []network.TaggedMessageHandler{
		{Tag: protocol.AgreementVoteTag, MessageHandler: network.HandlerFunc(i.processVoteMessage)},
		{Tag: protocol.ProposalPayloadTag, MessageHandler: network.HandlerFunc(i.processProposalMessage)},
		{Tag: protocol.VoteBundleTag, MessageHandler: network.HandlerFunc(i.processBundleMessage)},
	}
	net.RegisterHandlers(handlers)
	return i
}

func (i *networkImplSimulate) processVoteMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.voteCh)
}

func (i *networkImplSimulate) processProposalMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.proposalCh)
}

func (i *networkImplSimulate) processBundleMessage(raw network.IncomingMessage) network.OutgoingMessage {
	return i.processMessage(raw, i.bundleCh)
}

// i.e. process<Type>Message
func (i *networkImplSimulate) processMessage(raw network.IncomingMessage, submit chan<- Message) network.OutgoingMessage {
	metadata := &messageMetadata{raw: raw}

	select {
	case submit <- Message{MessageHandle: MessageHandle(metadata), Data: raw.Data}:
		// It would be slightly better to measure at de-queue
		// time, but that happens in many places in code and
		// this is much easier.
		messagesHandled.Inc(nil)
	default:
		messagesDropped.Inc(nil)
	}

	// Immediately ignore everything here, sometimes Relay/Broadcast/Disconnect later based on API handles saved from IncomingMessage
	return network.OutgoingMessage{Action: network.Ignore}
}

func (i *networkImplSimulate) Messages(t protocol.Tag) <-chan Message {
	switch t {
	case protocol.AgreementVoteTag:
		return i.voteCh
	case protocol.ProposalPayloadTag:
		return i.proposalCh
	case protocol.VoteBundleTag:
		return i.bundleCh
	default:
		logging.Base().Panicf("bad tag! %v", t)
		return nil
	}
}

func (i *networkImplSimulate) Broadcast(t protocol.Tag, data []byte) (err error) {
	err = i.net.Broadcast(context.Background(), t, data, false, nil)
	if err != nil {
		logging.Base().Infof("agreement: could not broadcast message with tag %v: %v", t, err)
	}
	return
}

func (i *networkImplSimulate) Relay(h MessageHandle, t protocol.Tag, data []byte) (err error) {
	msg := network.IncomingMessage{
	}
	msg.Tag = t
	msg.Data = data
	switch msg.Tag{
	case protocol.AgreementVoteTag:
		i.voteCh <- Message{MessageHandle: nil, Data: msg.Data}
	case protocol.ProposalPayloadTag:
		i.proposalCh <- Message{MessageHandle: nil, Data: msg.Data}
	}
	return

	//metadata := messageMetadataFromHandle(h)
	//if metadata == nil { // synthentic loopback
	//	err = i.net.Broadcast(context.Background(), t, data, false, nil)
	//	if err != nil {
	//		logging.Base().Infof("agreement: could not (pseudo)relay message with tag %v: %v", t, err)
	//	}
	//} else {
	//	err = i.net.Relay(context.Background(), t, data, false, metadata.raw.Sender)
	//	if err != nil {
	//		logging.Base().Infof("agreement: could not relay message from %v with tag %v: %v", metadata.raw.Sender, t, err)
	//	}
	//}
	//return
}

func (i *networkImplSimulate) Disconnect(h MessageHandle) {
	metadata := messageMetadataFromHandle(h)

	if metadata == nil { // synthentic loopback
		// TODO warn
		return
	}

	i.net.Disconnect(metadata.raw.Sender)
}

// broadcastTimeout is currently only used by test code.
// In test code we want to queue up a bunch of outbound packets and then see that they got through, so we need to wait at least a little bit for them to all go out.
// Normal agreement state machine code uses GossipNode.Broadcast non-blocking and may drop outbound packets.
func (i *networkImplSimulate) broadcastTimeout(t protocol.Tag, data []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return i.net.Broadcast(ctx, t, data, true, nil)
}

