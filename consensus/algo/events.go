package algo

type eventType int

// An event represents the communication of an event to a state machine.
//
// The eventType of the event corresponds to its semantics.  Metadata associated
// with an event is returned in the struct that implements the event interface.
type event interface {
	// t returns the eventType associated with the event.
	t() eventType

	// String returns a string description of an event.
	String() string

	// ComparableStr returns a comparable string description of an event
	// for testing purposes.
	ComparableStr() string
}

type externalEvent interface {
	event

	// ConsensusRound is the round related to this event.
	ConsensusRound() uint64

	// AttachConsensusVersion returns a copy of this externalEvent with a
	// ConsensusVersion attached.
	//AttachConsensusVersion(v ConsensusVersionView) externalEvent
}

const (
	// none is returned by state machines which have no event to return
	// otherwise.
	none eventType = iota

	// Some events originate from input sources to the agreement service.
	// These events are serialized via the demultiplexer.

	// votePresent, payloadPresent, and bundlePresent are emitted by the
	// network as input to the player state machine as messages are
	// received by the network.
	//
	// These events contain the unverfied version of the message object
	// itself as well as the MessageHandle tag.
	votePresent
	payloadPresent
	bundlePresent

	// voteVerified, payloadVerified, and bundleVerified are emitted by the
	// cryptoVerifier as input to the player state machine as cryptographic
	// verification completes for messages.
	//
	// These events contain the original unverified version of the message
	// object and the MessageHandle tag associated with the message when
	// first received.
	//
	// If verification has succeeded, these events also contain the verified
	// version of the message object, and their Err field is set to nil.  If
	// verification has failed, these events instead set the Err field with
	// the reason that verification failed.
	voteVerified
	payloadVerified
	bundleVerified

	// roundInterruption is emitted by the Ledger as input to the player
	// state machine when an external source observes that the player's
	// current round has completed concurrent with the player's operation.
	roundInterruption

	// timeout is emitted by the Clock as input to the player state machine
	// as the system observes that a timeout has been reached.
	//
	// The duration of the timeout is the one specified in player.Deadline.
	// This duration is expressed as an offset from the start of the current
	// period.
	//
	// fastTimeout is like timeout but for fast partition recovery.
	timeout
	fastTimeout

	// Other events are delivered from one state machine to another to
	// communicate some message or as a reply to some message.  These events
	// are internally dispatched via the router.

	// softThreshold, certThreshold, and nextThreshold are emitted by vote
	// state machines as they observe that a threshold of votes have been
	// met for a given step.
	//
	// These events may tell the player state machine to change their round,
	// their period, or possibly to send a cert vote.  These events are also
	// delivered to the proposal state machines to ensure that the correct
	// block is staged and relayed.
	softThreshold
	certThreshold
	nextThreshold

	// proposalCommittable is returned by the proposal state machines when a
	// proposal-value is observed to be committable (e.g., it is possible
	// that a certificate has formed for that proposal-value.
	proposalCommittable

	// proposalCommittable is returned by the proposal state machines when a
	// proposal-value is accepted.
	proposalAccepted

	// voteFiltered and voteMalformed are returned by the voteMachine and
	// the proposalMachine when a vote is invalid because it is corrupt
	// (voteMalformed) or irrelevant (voteFiltered).
	voteFiltered
	voteMalformed

	// bundleFiltered and bundleMalformed are returned by the voteMachine
	// when a bundle is invalid because it is corrupt (bundleMalformed) or
	// irrelevant (bundleFiltered).
	bundleFiltered
	bundleMalformed

	// payloadRejected and payloadMalformed are returned by the
	// proposalMachine when a proposal payload is invalid because it is
	// corrupt (payloadMalformed) or irrelevant (payloadRejected).
	payloadRejected
	payloadMalformed

	// payloadPipelined and payloadAccepted are returned by a proposal state
	// machine when either an unauthenticated (payloadPipelined) or an
	// authenticated (payloadAccepted) proposal payload is accepted and
	// stored.
	payloadPipelined
	payloadAccepted

	// proposalFrozen is sent between the player and proposal state machines
	// to specify that the proposal-vote with the lowest credential should
	// be fixed.
	proposalFrozen

	// voteAccepted is delivered from the voteMachine to its children after
	// a relevant vote has been validated.
	voteAccepted

	// newRound and newPeriod are delivered from the proposalMachine to
	// their children when a new round or period is observed.
	newRound
	newPeriod

	// readStaging is sent to the proposalPeriodMachine to read the staging
	// value for that period, if it exists.  It is returned by this machine
	// with the response.
	readStaging

	// readPinned is sent to the proposalStore to read the pinned value, if it exists.
	readPinned

	/*
	 * The following are event types that replace queries, and may warrant
	 * a revision to make them more state-machine-esque.
	 */

	// voteFilterRequest is an internal event emitted by vote aggregator and
	// the proposal manager to the vote step machines and the proposal period
	// machines respectively to check for duplicate votes. They enable the emission
	// of voteFilteredStep events.
	voteFilterRequest
	voteFilteredStep

	// nextThresholdStatusRequest is an internal event handled by voteMachinePeriod
	// that generates a corresponding nextThresholdStatus tracking whether the period
	// has seen none, a bot threshold, a value threshold, or both thresholds.
	nextThresholdStatusRequest
	nextThresholdStatus

	// freshestBundleRequest is an internal event handled by voteMachineRound that
	// generates a corresponding freshestBundle event.
	freshestBundleRequest
	freshestBundle

	// dumpVotesRequest is an internal event handled by voteTracker that generates
	// a corresponding dumpVotes event.
	dumpVotesRequest
	dumpVotes

	// For testing purposes only
	wrappedAction

	// checkpointReached indicates that we've completly persisted the agreement state to disk.
	// it's invoked by the end of the persistence loop on either success or failuire.
	checkpointReached
)
