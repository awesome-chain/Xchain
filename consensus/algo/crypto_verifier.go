package algo

import (
	"context"
)

type CryptoRequest struct {
	Message                   // the message we would like to verify.
	TaskIndex int             // Caller specific number that would be passed back in the cryptoResult.TaskIndex field
	Round     uint64           // The round that we're going to test against.
	Period    uint64          // The period associated with the message we're going to test.
	Pinned    bool            // A flag that is set if this is a pinned value for the given round.
	ctx       context.Context// A context for this request, if the context is cancelled then the request is stale.
}

type CryptoVerifier interface {
	Verify(ctx context.Context, request *CryptoRequest)

}

type PoolCryptoVerifier struct {
	//voteVerifier *AsyncVoteVerifier
	//
	//validator        BlockValidator
	//ledger           LedgerReader
	//proposalContexts pendingRequestsContext
	//log              logging.Logger
	//
	//quit chan struct{}
	//wg   sync.WaitGroup
}