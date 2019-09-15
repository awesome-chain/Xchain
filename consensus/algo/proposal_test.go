package algo

import (
	"github.com/awesome-chain/Xchain/consensus/ethash"
	"github.com/awesome-chain/Xchain/core"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/core/vm"
	"github.com/awesome-chain/Xchain/crypto"
	"github.com/awesome-chain/Xchain/crypto/vrf"
	"github.com/awesome-chain/Xchain/ethdb"
	"github.com/awesome-chain/Xchain/params"
	"math/big"
	"testing"
)

func TestProposal(t *testing.T) {
	// Configure and generate a sample block chain
	var (
		db      = ethdb.NewMemDatabase()
		key, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		address = crypto.PubkeyToAddress(key.PublicKey)
		funds   = big.NewInt(1000000000)
		gspec   = &core.Genesis{Config: params.TestChainConfig, Alloc: core.GenesisAlloc{address: {Balance: funds}}}
		genesis = gspec.MustCommit(db)
	)
	chain, _ := core.NewBlockChain(db, nil, gspec.Config, ethash.NewFaker(), vm.Config{})
	blocks, _ := core.GenerateChain(gspec.Config, genesis, ethash.NewFaker(), db, int(1), nil)
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to process block %d: %v", n, err)
	}
	var vrfSK vrf.PrivateKey
	vrfSK.PrivateKey = key
	header := &types.Header{
		Coinbase: address,
		Number:   new(big.Int).SetUint64(1),
	}
	newSeed, seedProof, err := DeriveNewSeed(address, &vrfSK, 1, 0, chain)
	if err != nil {
		t.Fatal(err)
	}
	header.Seed = newSeed
	//hash := header.HashNoSig()
	//sig, err := crypto.Sign(hash[:], vrfSK.PrivateKey)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//header.Sig = sig
	b := types.NewBlock(header, nil, nil, nil)
	p := MakeProposal(b, seedProof[:], 0, &key.PublicKey)
	VerifyNewSeed(p.UnauthenticatedProposal, chain)
	defer chain.Stop()
}
