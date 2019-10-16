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

package agreement

import (
	"context"
	"fmt"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/core/types"
	crypto2 "github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/awesome-chain/go-deadlock"
	"github.com/stretchr/testify/require"

	"github.com/awesome-chain/Xchain/consensus/algo/config"
	"github.com/awesome-chain/Xchain/consensus/algo/crypto"
	"github.com/awesome-chain/Xchain/consensus/algo/data/account"
	"github.com/awesome-chain/Xchain/consensus/algo/data/basics"
	"github.com/awesome-chain/Xchain/consensus/algo/data/committee"
	"github.com/awesome-chain/Xchain/consensus/algo/logging"
	"github.com/awesome-chain/Xchain/consensus/algo/protocol"
	"github.com/awesome-chain/Xchain/consensus/algo/util/db"
)

var poolAddr = basics.Address{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

var deadline = time.Second

var proto = protocol.ConsensusCurrentVersion

type signal struct {
	ch    chan struct{}
	fired bool
}

func makeSignal() signal {
	var s signal
	s.ch = make(chan struct{})
	return s
}

func (s signal) wait() {
	<-s.ch
}

func (s signal) fire() signal {
	if !s.fired {
		close(s.ch)
	}
	return signal{s.ch, true}
}

type testValidatedBlock struct {
	Inside Block
}

func (b testValidatedBlock) GetBlock() Block {
	return b.Inside
}

func (b testValidatedBlock) WithSeed(s committee.Seed) ValidatedBlock {
	//b.Inside.Header().Seed = common.Seed(s)
	return b
}

type testBlockValidator struct{}

func (v testBlockValidator) Validate(ctx context.Context, e Block) (ValidatedBlock, error) {
	return testValidatedBlock{Inside: e}, nil
}

type testBlockFactory struct {
	Owner int
}

func (f testBlockFactory) AssembleBlock(r basics.Round, deadline time.Time) (ValidatedBlock, error) {
	//return testValidatedBlock{Inside: Block{Hea: BlockHeader{Round: r}}}, nil
	return nil, nil
}

func (f testBlockFactory) AssembleProposal(vrfSK *vrf.PrivateKey, round basics.Round, per period, leger LedgerReader) (*proposal, *proposalValue, error) {
	address := crypto2.PubkeyToAddress(vrfSK.PrivateKey.PublicKey)
	newSeed, seedProof, err := DeriveNewSeed2(address, vrfSK, round, per, leger)
	if err != nil {
		return nil, nil, err
	}
	header := &types.Header{
		Coinbase: address,
		Number:   new(big.Int).SetUint64(uint64(round)),
	}

	header.Seed = common.Seed(newSeed)
	b := types.NewBlock(header, nil, nil, nil)
	vb := Block{
		Block: b,
	}
	var pubKey crypto2.S256PublicKey
	copy(pubKey[:], crypto2.FromECDSAPub(&vrfSK.PrivateKey.PublicKey))
	p := makeProposal2(vb, seedProof, per, address, pubKey)
	pv := p.value()
	return &p, &pv, nil
}

// If we try to read from high rounds, we panic and do not emit an error to find bugs during testing.
type testLedger struct {
	mu deadlock.Mutex

	entries   map[basics.Round]Block
	certs     map[basics.Round]Certificate
	nextRound basics.Round

	// constant
	state map[basics.Address]basics.BalanceRecord

	notifications map[basics.Round]signal
}

func makeTestLedger(state map[basics.Address]basics.BalanceRecord) Ledger {
	l := new(testLedger)
	l.entries = make(map[basics.Round]Block)
	l.certs = make(map[basics.Round]Certificate)
	l.nextRound = 1
	l.state = state
	l.notifications = make(map[basics.Round]signal)
	return l
}

func (l *testLedger) copy() *testLedger {
	dup := new(testLedger)

	dup.entries = make(map[basics.Round]Block)
	dup.certs = make(map[basics.Round]Certificate)
	dup.state = make(map[basics.Address]basics.BalanceRecord)
	dup.notifications = make(map[basics.Round]signal)

	for k, v := range l.entries {
		dup.entries[k] = v
	}
	for k, v := range l.certs {
		dup.certs[k] = v
	}
	for k, v := range l.state {
		dup.state[k] = v
	}
	for k, v := range dup.notifications {
		// note that old opened channels will now fire when these are closed
		dup.notifications[k] = v
	}
	dup.nextRound = l.nextRound

	return dup
}

func (l *testLedger) NextRound() basics.Round {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.nextRound
}

func (l *testLedger) Wait(r basics.Round) chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.notifications[r]; !ok {
		l.notifications[r] = makeSignal()
	}

	if l.nextRound > r {
		l.notify(r)
	}

	return l.notifications[r].ch
}

// note: this must be called when any new entry is written
// this should be called while the lock l.mu is held
func (l *testLedger) notify(r basics.Round) {
	if _, ok := l.notifications[r]; !ok {
		l.notifications[r] = makeSignal()
	}

	l.notifications[r] = l.notifications[r].fire()
}

func (l *testLedger) Seed(r basics.Round) (committee.Seed, error) {
	return committee.Seed{}, nil
	//l.mu.Lock()
	//defer l.mu.Unlock()
	//
	//if r >= l.nextRound {
	//	err := fmt.Errorf("Seed called on future round: %v > %v! (this is probably a bug)", r, l.nextRound)
	//	panic(err)
	//}
	//
	//b := l.entries[r]
	//return b.Seed(), nil
}

func (l *testLedger) LookupDigest(r basics.Round) (crypto.Digest, error) {
	return crypto.Digest{}, nil
	//l.mu.Lock()
	//defer l.mu.Unlock()
	//
	//if r >= l.nextRound {
	//	err := fmt.Errorf("Seed called on future round: %v > %v! (this is probably a bug)", r, l.nextRound)
	//	panic(err)
	//}
	//
	//return l.entries[r].Digest(), nil
}

func (l *testLedger) BalanceRecord(r basics.Round, a basics.Address) (basics.BalanceRecord, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if r >= l.nextRound {
		err := fmt.Errorf("BalanceRecord called on future round: %v > %v! (this is probably a bug)", r, l.nextRound)
		panic(err)
	}
	return l.state[a], nil
}

func (l *testLedger) BalanceRecord2(r basics.Round, a common.Address) (basics.BalanceRecord, error) {
	return basics.BalanceRecord{}, nil
}

func (l *testLedger) Circulation(r basics.Round) (basics.MicroAlgos, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if r >= l.nextRound {
		err := fmt.Errorf("Circulation called on future round: %v > %v! (this is probably a bug)", r, l.nextRound)
		panic(err)
	}

	var sum basics.MicroAlgos
	var overflowed bool
	for _, rec := range l.state {
		sum, overflowed = basics.OAddA(sum, rec.VotingStake())
		if overflowed {
			panic("circulation computation overflowed")
		}
	}
	return sum, nil
}

func (l *testLedger) ConsensusParams(basics.Round) (config.ConsensusParams, error) {
	return config.Consensus[protocol.ConsensusCurrentVersion], nil
}

func (l *testLedger) ConsensusVersion(basics.Round) (protocol.ConsensusVersion, error) {
	return protocol.ConsensusCurrentVersion, nil
}

func (l *testLedger) EnsureValidatedBlock(e ValidatedBlock, c Certificate) {
	l.EnsureBlock(e.GetBlock(), c)
}

func (l *testLedger) EnsureBlock(e Block, c Certificate) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.entries[e.Round()]; ok {
		if l.entries[e.Round()].Digest() != e.Digest() {
			err := fmt.Errorf("testLedger.EnsureBlock called with conflicting entries in round %v", e.Round())
			panic(err)
		}
	}

	l.entries[e.Round()] = e
	l.certs[e.Round()] = c

	if l.nextRound < e.Round()+1 {
		l.nextRound = e.Round() + 1
	}

	l.notify(e.Round())
}

func (l *testLedger) EnsureDigest(c Certificate, quit chan struct{}, verifier *AsyncVoteVerifier) {
	r := c.Round
	consistencyCheck := func() bool {
		l.mu.Lock()
		defer l.mu.Unlock()

		if r < l.nextRound {
			if l.entries[r].Digest() != c.Proposal.BlockDigest {
				err := fmt.Errorf("testLedger.EnsureDigest called with conflicting entries in round %v", r)
				panic(err)
			}
			return true
		}
		return false
	}

	if consistencyCheck() {
		return
	}

	select {
	case <-quit:
		return
	case <-l.Wait(r):
		if !consistencyCheck() {
			err := fmt.Errorf("Wait channel fired without matching block in round %v", r)
			panic(err)
		}
	}
}

func TestSimulate(t *testing.T) {
	f, _ := os.Create(t.Name() + ".log")
	logging.Base().SetJSONFormatter()
	logging.Base().SetOutput(f)
	logging.Base().SetLevel(logging.Debug)

	numAccounts := 1
	maxMoneyAtStart := 100001 // max money start
	minMoneyAtStart := 100000 // max money start
	E := basics.Round(50)     // max round

	// generate accounts
	genesis := make(map[basics.Address]basics.BalanceRecord)
	incentivePoolAtStart := uint64(1000 * 1000)
	accData := basics.MakeAccountData(basics.NotParticipating, basics.MicroAlgos{Raw: incentivePoolAtStart})
	genesis[poolAddr] = basics.BalanceRecord{Addr: poolAddr, AccountData: accData}
	gen := rand.New(rand.NewSource(2))

	_, accs, release := generateNAccounts(t, numAccounts, 0, E, minMoneyAtStart)
	defer release()
	for _, account := range accs {
		amount := basics.MicroAlgos{Raw: uint64(minMoneyAtStart + (gen.Int() % (maxMoneyAtStart - minMoneyAtStart)))}
		genesis[account.Address()] = basics.BalanceRecord{
			Addr: account.Address(),
			AccountData: basics.AccountData{
				Status:      basics.Online,
				MicroAlgos:  amount,
				SelectionID: account.VRFSecrets().PK,
				VoteID:      account.VotingSecrets().OneTimeSignatureVerifier,
			},
		}
	}

	l := makeTestLedger(genesis)
	err := Simulate(t.Name(), 10, deadline, l, SimpleKeyManager(accs), testBlockFactory{}, testBlockValidator{}, logging.Base())
	require.NoError(t, err)
}

func generateNAccounts(t *testing.T, N int, firstRound, lastRound basics.Round, fee int) (roots []account.Root, accounts []account.Participation, release func()) {
	allocatedAccessors := []db.Accessor{}
	release = func() {
		for _, acc := range allocatedAccessors {
			acc.Close()
		}
	}
	for i := 0; i < N; i++ {
		access, err := db.MakeAccessor(t.Name()+"_root_testingenv_"+strconv.Itoa(i), false, true)
		if err != nil {
			panic(err)
		}
		allocatedAccessors = append(allocatedAccessors, access)
		root, err := account.GenerateRoot(access)
		if err != nil {
			panic(err)
		}
		roots = append(roots, root)

		access, err = db.MakeAccessor(t.Name()+"_part_testingenv_"+strconv.Itoa(i), false, true)
		if err != nil {
			panic(err)
		}
		allocatedAccessors = append(allocatedAccessors, access)
		part, err := account.FillDBWithParticipationKeys(access, root.Address(), firstRound, lastRound, config.Consensus[protocol.ConsensusCurrentVersion].DefaultKeyDilution)
		if err != nil {
			panic(err)
		}
		accounts = append(accounts, part)
	}
	return
}
