package algo

import (
	"context"
	"github.com/awesome-chain/Xchain/util/execpool"
	"sync"
)

// AsyncVoteVerifier uses workers to verify agreement protocol votes and writes the results on an output channel specified by the user.
type AsyncVoteVerifier struct {
	done            chan struct{}
	wg              sync.WaitGroup
	workerWaitCh    chan struct{}
	backlogExecPool execpool.BacklogPool
	execpoolOut     chan interface{}
	ctx             context.Context
	ctxCancel       context.CancelFunc
}
