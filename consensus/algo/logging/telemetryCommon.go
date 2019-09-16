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

package logging

import (
	"sync"
	"time"

	"github.com/awesome-chain/go-deadlock"
	"github.com/sirupsen/logrus"

	"github.com/awesome-chain/Xchain/consensus/algo/logging/telemetryspec"
)

// TelemetryOperation wraps the context for an ongoing telemetry.StartOperation call
type TelemetryOperation struct {
	startTime      time.Time
	category       telemetryspec.Category
	identifier     telemetryspec.Operation
	telemetryState *telemetryState
	pending        int32
}

type telemetryState struct {
	history *logBuffer
	hook    *asyncTelemetryHook
}

// TelemetryConfig represents the configuration of Telemetry logging
type TelemetryConfig struct {
	Enable             bool
	URI                string
	Name               string
	GUID               string
	MinLogLevel        logrus.Level
	ReportHistoryLevel logrus.Level
	LogHistoryDepth    uint
	FilePath           string // Path to file on disk, if any
	ChainID            string `json:"-"`
	SessionGUID        string `json:"-"`
	UserName           string
	Password           string
}

type asyncTelemetryHook struct {
	deadlock.Mutex
	wrappedHook   logrus.Hook
	wg            sync.WaitGroup
	pending       []*logrus.Entry
	entries       chan *logrus.Entry
	quit          chan struct{}
	maxQueueDepth int
}

type hookFactory func(cfg TelemetryConfig) (logrus.Hook, error)
