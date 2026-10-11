// Package keys creates, wraps and unwraps scope keys (docs/sync-protocol.md
// §9). A scope key K(S,e) is 32 random bytes for XChaCha20-Poly1305; it is
// wrapped with age for every device that may read S, and for the root key.
package keys

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"strconv"

	"filippo.io/age"
	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// Size is the length of a scope key.
const Size = chacha20poly1305.KeySize

// ErrCheckMismatch is returned when an unwrapped key does not match the
// commitment in the ACL.
var ErrCheckMismatch = errors.New("keys: scope key does not match the ACL key check")

// New returns a fresh random scope key.
func New() ([]byte, error) {
	k := make([]byte, Size)
	if _, err := rand.Read(k); err != nil {
		return nil, err
	}
	return k, nil
}

// Check returns the commitment to K(S,e) published in the ACL's key_checks:
// "sha256:<b64>" of "ocm.key-check.v1\n<scope>\n<epoch>\n" || key.
//
// This is an addition to §8.1/§9. Key files are not signed, and a sync client
// replaces files that another client wrote under the same name, so anyone
// with write access to the folder could swap keys/<S>/<e>/<device>.age for a
// key they know. The device would then encrypt its segments with that key.
// Comparing the unwrapped key with the signed commitment in the ACL turns
// this into a detectable availability problem. Publishing the hash is safe:
// the key is 256 random bits.
func Check(scope db.Scope, epoch uint64, key []byte) string {
	h := sha256.New()
	io.WriteString(h, "ocm.key-check.v1\n"+string(scope)+"\n"+strconv.FormatUint(epoch, 10)+"\n")
	h.Write(key)
	var sum [32]byte
	h.Sum(sum[:0])
	return wire.FormatHash(sum)
}

// Verify compares key with the commitment from the ACL.
func Verify(scope db.Scope, epoch uint64, key []byte, check string) error {
	if len(key) != Size || subtle.ConstantTimeCompare([]byte(Check(scope, epoch, key)), []byte(check)) != 1 {
		return ErrCheckMismatch
	}
	return nil
}

// Wrap encrypts key for one age X25519 recipient ("age1...").
func Wrap(key []byte, recipient string) ([]byte, error) {
	r, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return nil, fmt.Errorf("keys: recipient: %w", err)
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, r)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(key); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unwrap decrypts a wrapped key with an age identity.
func Unwrap(wrapped []byte, id age.Identity) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(wrapped), id)
	if err != nil {
		return nil, fmt.Errorf("keys: unwrap: %w", err)
	}
	k, err := io.ReadAll(io.LimitReader(r, Size+1))
	if err != nil {
		return nil, fmt.Errorf("keys: unwrap: %w", err)
	}
	if len(k) != Size {
		return nil, fmt.Errorf("keys: unwrapped key has %d bytes", len(k))
	}
	return k, nil
}
