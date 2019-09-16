package algo

import "context"

type Service struct {
	quit   chan struct{}
	done   chan struct{}
	quitFn context.CancelFunc
	loopback *PseudoNode

}

