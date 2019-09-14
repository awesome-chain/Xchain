package algo

import (
	"github.com/awesome-chain/Xchain/accounts"
	"github.com/awesome-chain/Xchain/common"
	"github.com/awesome-chain/Xchain/consensus"
	"github.com/awesome-chain/Xchain/core/types"
	"github.com/awesome-chain/Xchain/crypto/sha3"
	"github.com/awesome-chain/Xchain/rlp"
	"math/big"
	"sync"
)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// SignTxFn is a signTx
type SignTxFn func(accounts.Account, *types.Transaction, *big.Int) (*types.Transaction, error)

// Algorand is the pure-proof-of-stake consensus engine.
type Algorand struct {
	sync.RWMutex
	signer   common.Address
	signFn   SignerFn // Signer function to authorize hashes with
	signTxFn SignTxFn // Sign transaction function to sign tx
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (a *Algorand) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()
	// Sealing the genesis block is not supported
	//number := header.Number.Uint64()
	//if number == 0 {
	//	return nil, errUnknownBlock
	//}
	a.RLock()
	signer, signFn := a.signer, a.signFn
	a.RUnlock()
	//select {
	//case <-stop:
	//	return nil, nil
	//case <-time.After(delay):
	//}
	hash := HashHeader(header)
	sig, err := signFn(accounts.Account{Address: signer}, hash.Bytes())
	if err != nil {
		return nil, err
	}
	header.Sig = sig
	return block.WithSeal(header), nil
}

func HashHeader(h *types.Header) common.Hash {
	return rlpHash([]interface{}{
		h.ParentHash,
		h.UncleHash,
		h.Coinbase,
		h.Root,
		h.TxHash,
		h.ReceiptHash,
		h.Seed,
		h.Bloom,
		h.Difficulty,
		h.Number,
		h.GasLimit,
		h.GasUsed,
		h.Time,
		h.Extra,
	})
}

// ecrecover extracts the Ethereum account address from a signed header.
//func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
//	// If the signature's already cached, return that
//	hash := header.Hash()
//	if address, known := sigcache.Get(hash); known {
//		return address.(common.Address), nil
//	}
//	// Retrieve the signature from the header extra-data
//	if len(header.Extra) < extraSeal {
//		return common.Address{}, errMissingSignature
//	}
//	signature := header.Extra[len(header.Extra)-extraSeal:]
//
//	// Recover the public key and the Ethereum address
//	headerSigHash, err := sigHash(header)
//	if err != nil {
//		return common.Address{}, err
//	}
//	pubkey, err := crypto.Ecrecover(headerSigHash.Bytes(), signature)
//	if err != nil {
//		return common.Address{}, err
//	}
//	var signer common.Address
//	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])
//
//	sigcache.Add(hash, signer)
//	return signer, nil
//}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}
