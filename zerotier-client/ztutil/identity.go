package ztutil

import (
	secrand "crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/salsa20/salsa"

	"golang.org/x/crypto/curve25519"

	"golang.org/x/crypto/ed25519"
)

const ztIdentityGenMemory = 2097152
const ztIdentityHashCashFirstByteLessThan = 17

func computeZeroTierIdentityMemoryHardHash(publicKey []byte) []byte {
	s512 := sha512.Sum512(publicKey)

	var genmem [ztIdentityGenMemory]byte
	var s20key [32]byte
	var s20ctr [16]byte
	var s20ctri uint64
	copy(s20key[:], s512[0:32])
	copy(s20ctr[0:8], s512[32:40])
	salsa.XORKeyStream(genmem[0:64], genmem[0:64], &s20ctr, &s20key)
	s20ctri++
	for i := 64; i < ztIdentityGenMemory; i += 64 {
		binary.LittleEndian.PutUint64(s20ctr[8:16], s20ctri)
		salsa.XORKeyStream(genmem[i:i+64], genmem[i-64:i], &s20ctr, &s20key)
		s20ctri++
	}

	var tmp [8]byte
	for i := 0; i < ztIdentityGenMemory; {
		idx1 := uint(binary.BigEndian.Uint64(genmem[i:])&7) * 8
		i += 8
		idx2 := (uint(binary.BigEndian.Uint64(genmem[i:])) % uint(ztIdentityGenMemory/8)) * 8
		i += 8
		gm := genmem[idx2 : idx2+8]
		d := s512[idx1 : idx1+8]
		copy(tmp[:], gm)
		copy(gm, d)
		copy(d, tmp[:])
		binary.LittleEndian.PutUint64(s20ctr[8:16], s20ctri)
		salsa.XORKeyStream(s512[:], s512[:], &s20ctr, &s20key)
		s20ctri++
	}

	return s512[:]
}

// generateDualPair generates a key pair containing two pairs: one for curve25519 and one for ed25519.
func generateDualPair() (pub [64]byte, priv [64]byte) {
	k0pub, k0priv, _ := ed25519.GenerateKey(secrand.Reader)
	var k1pub, k1priv [32]byte
	secrand.Read(k1priv[:])
	curve25519.ScalarBaseMult(&k1pub, &k1priv)
	copy(pub[0:32], k1pub[:])
	copy(pub[32:64], k0pub[0:32])
	copy(priv[0:32], k1priv[:])
	copy(priv[32:64], k0priv[0:32])
	return
}

// ZeroTierIdentity contains a public key, a private key, and a string representation of the identity.
type ZeroTierIdentity struct {
	address    uint64 // ZeroTier address, only least significant 40 bits are used
	publicKey  [64]byte
	privateKey *[64]byte
}

// NewZeroTierIdentity creates a new ZeroTier Identity.
// This can be a little bit time consuming due to one way proof of work requirements (usually a few hundred milliseconds).
func NewZeroTierIdentity() (id ZeroTierIdentity) {
	for {
		pub, priv := generateDualPair()
		dig := computeZeroTierIdentityMemoryHardHash(pub[:])
		if dig[0] < ztIdentityHashCashFirstByteLessThan && dig[59] != 0xff {
			id.address = uint64(dig[59])
			id.address <<= 8
			id.address |= uint64(dig[60])
			id.address <<= 8
			id.address |= uint64(dig[61])
			id.address <<= 8
			id.address |= uint64(dig[62])
			id.address <<= 8
			id.address |= uint64(dig[63])
			if id.address != 0 {
				id.publicKey = pub
				id.privateKey = &priv
				break
			}
		}
	}
	return
}

// PrivateKeyString returns the full identity.secret if the private key is set, or an empty string if no private key is set.
func (id *ZeroTierIdentity) PrivateKeyString() string {
	if id.privateKey != nil {
		s := fmt.Sprintf("%.10x:0:%x:%x", id.address, id.publicKey, *id.privateKey)
		return s
	}
	return ""
}

// PublicKeyString returns identity.public contents.
func (id *ZeroTierIdentity) PublicKeyString() string {
	s := fmt.Sprintf("%.10x:0:%x", id.address, id.publicKey)
	return s
}

// IDString returns the NodeID as a 10-digit hex string
func (id *ZeroTierIdentity) IDString() string {
	s := fmt.Sprintf("%.10x", id.address)
	return s
}

// ID returns the ZeroTier address as a uint64
func (id *ZeroTierIdentity) ID() uint64 {
	return id.address
}

// PrivateKey returns the bytes of the private key (or nil if not set)
func (id *ZeroTierIdentity) PrivateKey() *[64]byte {
	return id.privateKey
}

// PublicKey returns the public key bytes
func (id *ZeroTierIdentity) PublicKey() [64]byte {
	return id.publicKey
}
