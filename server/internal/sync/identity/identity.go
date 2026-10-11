// Package identity holds the key pairs of a device and of a club's root key
// (docs/sync-protocol.md §5).
package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

var deviceIDEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// DeviceID derives the device id of §5.1: the first 16 bytes of
// SHA-256(sign_pub) as lowercase base32 without padding (26 characters).
func DeviceID(signPub ed25519.PublicKey) string {
	sum := sha256.Sum256(signPub)
	return strings.ToLower(deviceIDEncoding.EncodeToString(sum[:16]))
}

// KeyPair is an Ed25519 signing key plus an age X25519 identity. Devices and
// the root key both consist of one.
type KeyPair struct {
	Sign ed25519.PrivateKey
	Age  *age.X25519Identity
}

// Generate creates a fresh key pair from the OS CSPRNG.
func Generate() (*KeyPair, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate signing key: %w", err)
	}
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate age identity: %w", err)
	}
	return &KeyPair{Sign: priv, Age: id}, nil
}

// SignPub returns the public signing key.
func (k *KeyPair) SignPub() ed25519.PublicKey { return k.Sign.Public().(ed25519.PublicKey) }

// SignPubB64 returns the public signing key in its b64 form.
func (k *KeyPair) SignPubB64() string { return wire.EncodeB64(k.SignPub()) }

// AgeRecipient returns the public age recipient ("age1...").
func (k *KeyPair) AgeRecipient() string { return k.Age.Recipient().String() }

// Wipe overwrites the private signing key in memory. The age identity keeps
// its scalar in an unexported field that cannot be cleared; dropping the
// reference is all that can be done for it.
func (k *KeyPair) Wipe() {
	clear(k.Sign)
	k.Sign = nil
	k.Age = nil
}

// Device is the identity of this installation.
type Device struct {
	ID string
	*KeyPair
}

// NewDevice wraps a key pair and derives its id.
func NewDevice(k *KeyPair) *Device { return &Device{ID: DeviceID(k.SignPub()), KeyPair: k} }

// SafetyCode is the 6-digit code of §17.2 that admin and new member compare
// out of band. It is derived from SHA-256(sign_pub || age_recipient); the
// spec does not fix the mapping to digits, so the first 8 bytes of the hash
// are read as a big-endian integer modulo 10^6 (bias below 2^-40).
func SafetyCode(signPub ed25519.PublicKey, ageRecipient string) string {
	h := sha256.New()
	h.Write(signPub)
	h.Write([]byte(ageRecipient))
	sum := h.Sum(nil)
	var n uint64
	for _, b := range sum[:8] {
		n = n<<8 | uint64(b)
	}
	return fmt.Sprintf("%06d", n%1_000_000)
}
