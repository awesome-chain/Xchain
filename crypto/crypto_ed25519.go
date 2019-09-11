package crypto

import (
	"golang.org/x/crypto/ed25519"
	"math/big"
)

// ToECDSA creates a private key with the given D value.
func ToECD25519(d []byte) (crypto.Priva, error) {
	return toECDSA(d, true)
}