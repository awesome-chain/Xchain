package ed25519

// #cgo CFLAGS: -Wall -std=c99
// #cgo CFLAGS: -I${SRCDIR}/include/
// #cgo LDFLAGS: ${SRCDIR}/lib/libsodium.a
// #include <stdint.h>
// #include "sodium.h"
import "C"

import (
	"crypto/rand"
	"crypto/sha512"
	"errors"
)

// DigestSize is the number of bytes in the preferred hash Digest used here.
const (
	DigestSize     = sha512.Size256
	multiSigString = "multisigaddr"
	maxMultisig    = 255
)

var (
	ErrInvalidVersion           = errors.New("invalid version")
	ErrInvalidAddress           = errors.New("invalid address")
	ErrInvalidThreshold         = errors.New("invalid threshold")
	ErrInvalidNumberOfSignature = errors.New("invalid number of signatures")
	ErrKeyNotExist              = errors.New("key does not exist")
	ErrSubSigVerification       = errors.New("verification failure: subsignature")
	ErrKeysNotMatch             = errors.New("public key lists do not match")
	ErrInvalidDuplicates        = errors.New("invalid duplicates")
	ErrUnknownVersion           = errors.New("unknown version")
)

// Digest represents a 32-byte value holding the 256-bit Hash digest.
type Digest [DigestSize]byte

type (
	// A VrfPrivkey is a private key used for producing VRF proofs.
	// Specifically, we use a 64-byte ed25519 private key (the latter 32-bytes are the precomputed public key)
	VrfPrivkey [64]uint8
	// A VrfPubkey is a public key that can be used to verify VRF proofs.
	VrfPubkey [32]uint8
	// A VrfProof for a message can be generated with a secret key and verified against a public key, like a signature.
	// Proofs are malleable, however, for a given message and public key, the VRF output that can be computed from a proof is unique.
	VrfProof [80]uint8
	// VrfOutput is a 64-byte pseudorandom value that can be computed from a VrfProof.
	// The VRF scheme guarantees that such output will be unique
	VrfOutput  [64]uint8
	PublicKey  [32]byte
	PrivateKey [64]byte
	Signature  [64]byte
)

// MultiSignatures is the structure that holds multiple Sub Sigs
type MultiSignatures struct {
	Version       uint8
	Threshold     uint8
	SubSignatures []*MultiSubSig
}

// MultiSubSig is a struct that holds a pair of public key and signatures
type MultiSubSig struct {
	Key PublicKey
	Sig Signature
}

func ED25519GenerateKey() (public PublicKey, secret PrivateKey, err error) {
	var seed [32]byte
	_, err = rand.Read(seed[:])
	if err != nil {
		return
	}
	C.crypto_sign_ed25519_seed_keypair((*C.uchar)(&public[0]), (*C.uchar)(&secret[0]), (*C.uchar)(&seed[0]))
	return

}

func ED25519Sign(secret PrivateKey, data []byte) (sig Signature) {
	// &data[0] will make Go panic if msg is zero length
	d := (*C.uchar)(C.NULL)
	if len(data) != 0 {
		d = (*C.uchar)(&data[0])
	}
	// https://download.libsodium.org/doc/public-key_cryptography/public-key_signatures#detached-mode
	C.crypto_sign_ed25519_detached((*C.uchar)(&sig[0]), (*C.ulonglong)(C.NULL), d, C.ulonglong(len(data)), (*C.uchar)(&secret[0]))
	return
}

func ED25519Verify(public PublicKey, data []byte, sig Signature) bool {
	// &data[0] will make Go panic if msg is zero length
	d := (*C.uchar)(C.NULL)
	if len(data) != 0 {
		d = (*C.uchar)(&data[0])
	}
	// https://download.libsodium.org/doc/public-key_cryptography/public-key_signatures#detached-mode
	result := C.crypto_sign_ed25519_verify_detached((*C.uchar)(&sig[0]), d, C.ulonglong(len(data)), (*C.uchar)(&public[0]))
	return result == 0
}

func VRFKeygen() (pub VrfPubkey, priv VrfPrivkey) {
	C.crypto_vrf_keypair((*C.uchar)(&pub[0]), (*C.uchar)(&priv[0]))
	return pub, priv
}

func RetrievePubkey(prvKey PrivateKey) (pubKey PublicKey) {
	C.crypto_vrf_sk_to_pk((*C.uchar)(&pubKey[0]), (*C.uchar)(&prvKey[0]))
	return
}

func VRFProve(prvKey VrfPrivkey, msg []byte) (proof VrfProof, ok bool) {
	m := (*C.uchar)(C.NULL)
	if len(msg) != 0 {
		m = (*C.uchar)(&msg[0])
	}
	ret := C.crypto_vrf_prove((*C.uchar)(&proof[0]), (*C.uchar)(&prvKey[0]), (*C.uchar)(m), (C.ulonglong)(len(msg)))
	return proof, ret == 0
}

func VRFVerify(pubKey VrfPubkey, proof VrfProof, msg []byte) (bool, VrfOutput) {
	var out VrfOutput
	// &msg[0] will make Go panic if msg is zero length
	m := (*C.uchar)(C.NULL)
	if len(msg) != 0 {
		m = (*C.uchar)(&msg[0])
	}
	ret := C.crypto_vrf_verify((*C.uchar)(&out[0]), (*C.uchar)(&pubKey[0]), (*C.uchar)(&proof[0]), (*C.uchar)(m), (C.ulonglong)(len(msg)))
	return ret == 0, out
}

func Hash(data []byte) Digest {
	return sha512.Sum512_256(data)
}

func MultiSigAddrGen(version, threshold uint8, pk []PublicKey) (addr Digest, err error) {
	if version != 1 {
		err = ErrUnknownVersion
		return
	}

	if threshold == 0 || len(pk) == 0 || int(threshold) > len(pk) {
		err = ErrInvalidThreshold
		return
	}

	buffer := append([]byte(multiSigString), byte(version), byte(threshold))
	for _, pki := range pk {
		buffer = append(buffer, pki[:]...)
	}
	return Hash(buffer), nil
}

// MultiSignWithOneKey is for each device individually signs the digest
func MultiSignWithOneKey(sigs *MultiSignatures, msg []byte, addr Digest, version, threshold uint8, pk []PublicKey, sk PrivateKey) error {
	if version != 1 {
		return ErrUnknownVersion
	}
	pubKey := PublicKey(RetrievePubkey([64]byte(sk)))
	// check the address matches the keys
	addrNew, err := MultiSigAddrGen(version, threshold, pk)
	if err != nil {
		return err
	}
	if addr != addrNew {
		return ErrInvalidAddress
	}
	keyExist := false
	for k, v := range sigs.SubSignatures {
		if v.Key == pubKey {
			keyExist = true
			sigs.SubSignatures[k].Sig = ED25519Sign(sk, msg)
			break
		}
	}
	if !keyExist {
		return ErrKeyNotExist
	}
	return nil
}

// MultisigVerify verifies an assembled MultisigSig
func MultiSignVerify(msg []byte, addr Digest, sigs *MultiSignatures) (bool, error) {
	if len(sigs.SubSignatures) == 0{
		return false, nil
	}
	pks := make([]PublicKey, 0)
	for _, v := range sigs.SubSignatures {
		pks = append(pks, v.Key)
	}
	// check the address matches the keys
	addrNew, err := MultiSigAddrGen(sigs.Version, sigs.Threshold, pks)
	if err != nil {
		return false, err
	}
	if addr != addrNew {
		return false, nil
	}

	// check that we don't have too many multisig subsigs
	if len(sigs.SubSignatures) > maxMultisig {
		return false, nil
	}

	// check that we don't have too few multisig subsigs
	if len(sigs.SubSignatures) < int(sigs.Threshold) {
		return false, nil
	}

	// checks the number of non-blank signatures is no less than threshold
	var counter uint8
	for _, v := range sigs.SubSignatures {
		if (v.Sig != Signature{}) {
			counter++
		}
	}
	if counter < sigs.Threshold {
		return false, nil
	}

	// checks individual signature verifies
	var verifiedCount int
	for _, v := range sigs.SubSignatures {
		if (v.Sig != Signature{}) {
			if !ED25519Verify(v.Key, msg, v.Sig) {
				return false, nil
			}
			verifiedCount++
		}
	}
	// sanity check. if we get here then every non-blank subsig should have
	// been verified successfully, and we should have had enough of them
	if verifiedCount < int(sigs.Threshold) {
		return false, nil
	}
	return true, nil
}
