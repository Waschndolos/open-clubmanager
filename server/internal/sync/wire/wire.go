// Package wire holds the encodings shared by the sync protocol files
// (docs/sync-protocol.md §3): base64url without padding for binary values,
// "sha256:<b64>" for hashes, and detached Ed25519 signatures over the exact
// bytes of a file.
package wire

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ErrBadSignature is returned when a detached signature does not verify.
var ErrBadSignature = errors.New("signature does not verify")

// B64 encodes binary values as base64url without padding.
var B64 = base64.RawURLEncoding

// EncodeB64 returns the b64 form of b.
func EncodeB64(b []byte) string { return B64.EncodeToString(b) }

// DecodeB64 parses a b64 value. Padding and the standard alphabet are
// rejected, so every value has exactly one encoding.
func DecodeB64(s string) ([]byte, error) {
	b, err := B64.Strict().DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("malformed b64 value: %w", err)
	}
	return b, nil
}

// HashPrefix is the prefix of every hash written in protocol files.
const HashPrefix = "sha256:"

// Hash returns "sha256:<b64>" of data.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return FormatHash(sum)
}

// FormatHash returns the "sha256:<b64>" form of a digest.
func FormatHash(sum [32]byte) string { return HashPrefix + EncodeB64(sum[:]) }

// ParseHash parses the "sha256:<b64>" form.
func ParseHash(s string) ([32]byte, error) {
	var out [32]byte
	rest, ok := strings.CutPrefix(s, HashPrefix)
	if !ok {
		return out, fmt.Errorf("hash %q lacks %q prefix", s, HashPrefix)
	}
	b, err := DecodeB64(rest)
	if err != nil {
		return out, err
	}
	if len(b) != len(out) {
		return out, fmt.Errorf("hash %q has %d bytes, want 32", s, len(b))
	}
	copy(out[:], b)
	return out, nil
}

// ParsePublicKey decodes a b64 Ed25519 public key.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := DecodeB64(s)
	if err != nil {
		return nil, err
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key has %d bytes, want %d", len(b), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(b), nil
}

// Sign returns the content of a ".sig" file: the b64 Ed25519 signature over
// the exact bytes of data.
func Sign(priv ed25519.PrivateKey, data []byte) []byte {
	return []byte(EncodeB64(ed25519.Sign(priv, data)))
}

// Verify checks the content of a ".sig" file against data. Surrounding white
// space in the signature file is ignored, since editors and sync clients may
// add a trailing newline.
func Verify(pub ed25519.PublicKey, data, sigFile []byte) error {
	sig, err := DecodeB64(string(bytes.TrimSpace(sigFile)))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrBadSignature
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, data, sig) {
		return ErrBadSignature
	}
	return nil
}

// DecodeStrict parses one JSON value into out, rejecting invalid UTF-8,
// duplicate object keys, unknown fields and trailing data (§3). Signed
// protocol files are parsed with it so that a file cannot carry content this
// version of the app would silently ignore or read differently than another
// implementation. Numbers decode as json.Number into interface values.
func DecodeStrict(data []byte, out any) error {
	if !utf8.Valid(data) {
		return errors.New("invalid UTF-8")
	}
	if err := checkDuplicateKeys(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing data after JSON object")
	}
	return nil
}

// checkDuplicateKeys walks the JSON tokens of data and fails if an object
// repeats a key. encoding/json would silently keep the last one.
func checkDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	type frame struct {
		object bool
		keys   map[string]bool
		key    bool // next token in an object is a key
	}
	var stack []*frame
	for {
		tok, err := d.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var top *frame
		if len(stack) > 0 {
			top = stack[len(stack)-1]
		}
		if top != nil && top.object && top.key {
			if k, ok := tok.(string); ok {
				if top.keys[k] {
					return fmt.Errorf("duplicate object key %q", k)
				}
				top.keys[k] = true
				top.key = false
				continue
			}
		}
		switch tok {
		case json.Delim('{'):
			stack = append(stack, &frame{object: true, keys: map[string]bool{}, key: true})
			continue
		case json.Delim('['):
			stack = append(stack, &frame{})
			continue
		case json.Delim('}'), json.Delim(']'):
			stack = stack[:len(stack)-1]
		}
		// A value ended: the next token of an enclosing object is a key.
		if len(stack) > 0 && stack[len(stack)-1].object {
			stack[len(stack)-1].key = true
		}
	}
}
