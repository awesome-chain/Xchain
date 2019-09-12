package algo

import (
	"crypto/sha512"
	"github.com/awesome-chain/Xchain/common"
)

// HashObj computes a hash of a Hashable object and its type
func HashObj(h Hashable) common.Hash {
	return Hash(hashRep(h))
}

// Hash computes the SHASum512_256 hash of an array of bytes
func Hash(data []byte) common.Hash {
	return sha512.Sum512_256(data)
}
