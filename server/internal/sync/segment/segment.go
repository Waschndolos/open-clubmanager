// Package segment implements the file format of change segments and
// snapshots (docs/sync-protocol.md §11, §15.1):
//
//	magic       4 bytes   "OCM1"
//	hdr_len     4 bytes   big-endian uint32
//	header      hdr_len bytes, UTF-8 JSON
//	ciphertext  remaining bytes minus 64
//	signature   64 bytes  Ed25519 over (magic || hdr_len || header || ciphertext)
//
// The ciphertext is XChaCha20-Poly1305 under K(scope, epoch) with the exact
// header bytes as associated data; the plaintext is gzip-compressed JSON.
// The package knows nothing about ACLs or the database: callers decide which
// key and which public key apply.
package segment

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// Limits of §11.2, enforced before allocating or decompressing.
const (
	Magic         = "OCM1"
	MaxHeader     = 64 << 10
	MaxCiphertext = 16 << 20
	MaxPlaintext  = 64 << 20

	prefixLen = 8
	sigLen    = ed25519.SignatureSize
)

// SnapshotType is the "type" of a snapshot header.
const SnapshotType = "snapshot"

var (
	// ErrMalformed is returned for files that violate the framing, the size
	// limits or the header schema. A file that is still being synced by a
	// cloud client usually fails here or at the signature.
	ErrMalformed = errors.New("segment: malformed")
	// ErrBadSignature is returned when the signature does not verify.
	ErrBadSignature = errors.New("segment: signature does not verify")
	// ErrDecrypt is returned when the AEAD tag does not verify.
	ErrDecrypt = errors.New("segment: decryption failed")
)

// Header is the header of a change segment (§11.1).
type Header struct {
	ClubID     string   `json:"club_id"`
	DeviceID   string   `json:"device_id"`
	Scope      db.Scope `json:"scope"`
	Seq        uint64   `json:"seq"`
	Prev       *string  `json:"prev"`
	Epoch      uint64   `json:"epoch"`
	ACLVersion uint64   `json:"acl_version"`
	Schema     int      `json:"schema"`
	Nonce      string   `json:"nonce"`
	HLCMin     string   `json:"hlc_min"`
	HLCMax     string   `json:"hlc_max"`
}

// Frontier is a stream position in a snapshot (§15.1): the last included
// seq and the complete-file hash of that segment (nil for seq 0).
type Frontier struct {
	Seq  uint64  `json:"seq"`
	Hash *string `json:"hash"`
}

// SnapshotHeader is the header of a snapshot (§15.1): a segment header with
// "type": "snapshot" and "streams" instead of seq/prev. The spec leaves the
// remaining fields open; HLC is the snapshot's timestamp, also in its file
// name.
type SnapshotHeader struct {
	Type       string              `json:"type"`
	ClubID     string              `json:"club_id"`
	DeviceID   string              `json:"device_id"`
	Scope      db.Scope            `json:"scope"`
	Streams    map[string]Frontier `json:"streams"`
	Epoch      uint64              `json:"epoch"`
	ACLVersion uint64              `json:"acl_version"`
	Schema     int                 `json:"schema"`
	Nonce      string              `json:"nonce"`
	HLC        string              `json:"hlc"`
}

// Frame is a parsed, not yet verified file.
type Frame struct {
	Raw        []byte
	Header     []byte
	Ciphertext []byte
	signed     []byte
	sig        []byte
}

// Hash returns "sha256:<b64>" of the complete file, as used in prev,
// cutoffs and snapshot frontiers.
func (f *Frame) Hash() string { return wire.Hash(f.Raw) }

// Parse checks the framing and the size limits of a file.
func Parse(raw []byte) (*Frame, error) {
	if len(raw) < prefixLen+sigLen || string(raw[:4]) != Magic {
		return nil, fmt.Errorf("%w: framing", ErrMalformed)
	}
	n := binary.BigEndian.Uint32(raw[4:8])
	if n == 0 || n > MaxHeader {
		return nil, fmt.Errorf("%w: header length %d", ErrMalformed, n)
	}
	if uint64(len(raw)) < uint64(prefixLen)+uint64(n)+sigLen {
		return nil, fmt.Errorf("%w: truncated", ErrMalformed)
	}
	end := len(raw) - sigLen
	ct := raw[prefixLen+int(n) : end]
	if len(ct) > MaxCiphertext {
		return nil, fmt.Errorf("%w: ciphertext of %d bytes", ErrMalformed, len(ct))
	}
	if len(ct) < chacha20poly1305.Overhead {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrMalformed)
	}
	return &Frame{
		Raw:        raw,
		Header:     raw[prefixLen : prefixLen+int(n)],
		Ciphertext: ct,
		signed:     raw[:end],
		sig:        raw[end:],
	}, nil
}

// Verify checks the signature with the author's public key.
func (f *Frame) Verify(pub ed25519.PublicKey) error {
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, f.signed, f.sig) {
		return ErrBadSignature
	}
	return nil
}

// SegmentHeader decodes the header as a change segment header.
func (f *Frame) SegmentHeader() (*Header, error) {
	var h Header
	if err := wire.DecodeStrict(f.Header, &h); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrMalformed, err)
	}
	if err := h.check(); err != nil {
		return nil, err
	}
	return &h, nil
}

func (h *Header) check() error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: header: "+format, append([]any{ErrMalformed}, args...)...)
	}
	switch {
	case !layout.ValidDeviceID(h.DeviceID):
		return bad("device_id")
	case !layout.ValidScope(h.Scope):
		return bad("scope")
	case h.Seq == 0:
		return bad("seq 0")
	case (h.Seq == 1) != (h.Prev == nil):
		return bad("prev must be null exactly for seq 1")
	case h.Epoch == 0 || h.ACLVersion == 0 || h.Schema < 1:
		return bad("epoch, acl_version or schema")
	}
	if h.Prev != nil {
		if _, err := wire.ParseHash(*h.Prev); err != nil {
			return bad("prev: %v", err)
		}
	}
	if err := checkNonce(h.Nonce); err != nil {
		return err
	}
	lo, err1 := hlc.Parse(h.HLCMin)
	hi, err2 := hlc.Parse(h.HLCMax)
	if err1 != nil || err2 != nil || lo.Compare(hi) > 0 {
		return bad("hlc range")
	}
	return nil
}

// SnapshotHeader decodes the header as a snapshot header.
func (f *Frame) SnapshotHeader() (*SnapshotHeader, error) {
	var h SnapshotHeader
	if err := wire.DecodeStrict(f.Header, &h); err != nil {
		return nil, fmt.Errorf("%w: header: %v", ErrMalformed, err)
	}
	bad := func(format string, args ...any) (*SnapshotHeader, error) {
		return nil, fmt.Errorf("%w: snapshot header: "+format, append([]any{ErrMalformed}, args...)...)
	}
	switch {
	case h.Type != SnapshotType:
		return bad("type %q", h.Type)
	case !layout.ValidDeviceID(h.DeviceID):
		return bad("device_id")
	case !layout.ValidScope(h.Scope):
		return bad("scope")
	case h.Epoch == 0 || h.ACLVersion == 0 || h.Schema < 1:
		return bad("epoch, acl_version or schema")
	case h.Streams == nil:
		return bad("streams missing")
	}
	if _, err := hlc.Parse(h.HLC); err != nil {
		return bad("hlc")
	}
	for id, fr := range h.Streams {
		if !layout.ValidDeviceID(id) {
			return bad("stream %q", id)
		}
		if (fr.Seq == 0) != (fr.Hash == nil) {
			return bad("frontier of %s", id)
		}
		if fr.Hash != nil {
			if _, err := wire.ParseHash(*fr.Hash); err != nil {
				return bad("frontier of %s: %v", id, err)
			}
		}
	}
	if err := checkNonce(h.Nonce); err != nil {
		return nil, err
	}
	return &h, nil
}

func checkNonce(s string) error {
	n, err := wire.DecodeB64(s)
	if err != nil || len(n) != chacha20poly1305.NonceSizeX {
		return fmt.Errorf("%w: header: nonce", ErrMalformed)
	}
	return nil
}

// Open decrypts the ciphertext with key, using the header bytes as
// associated data, and decompresses it. The nonce is taken from the header,
// which the caller has decoded and checked before.
func (f *Frame) Open(key []byte, nonceB64 string) ([]byte, error) {
	nonce, err := wire.DecodeB64(nonceB64)
	if err != nil || len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, fmt.Errorf("%w: nonce", ErrMalformed)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	gz, err := aead.Open(nil, nonce, f.Ciphertext, f.Header)
	if err != nil {
		return nil, ErrDecrypt
	}
	return gunzip(gz)
}

// gunzip decompresses data and fails once the output exceeds MaxPlaintext.
func gunzip(data []byte) ([]byte, error) {
	br := bytes.NewReader(data)
	r, err := gzip.NewReader(br)
	if err != nil {
		return nil, fmt.Errorf("%w: gzip: %v", ErrMalformed, err)
	}
	r.Multistream(false)
	out, err := io.ReadAll(io.LimitReader(r, MaxPlaintext+1))
	if err != nil {
		return nil, fmt.Errorf("%w: gzip: %v", ErrMalformed, err)
	}
	if len(out) > MaxPlaintext {
		return nil, fmt.Errorf("%w: plaintext exceeds %d bytes", ErrMalformed, MaxPlaintext)
	}
	// Trailing bytes after the gzip member would be data the signature
	// covers but no reader sees.
	if br.Len() > 0 {
		return nil, fmt.Errorf("%w: trailing data after gzip stream", ErrMalformed)
	}
	return out, nil
}

// NewNonce returns a fresh 24-byte nonce from the OS CSPRNG in b64 form
// (§11.2: never derived from seq or device_id).
func NewNonce() (string, error) {
	n := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(n); err != nil {
		return "", err
	}
	return wire.EncodeB64(n), nil
}

// Seal builds a complete file: it encodes header (whose nonce must be set,
// see NewNonce), compresses and encrypts plaintext under key with the header
// bytes as associated data, and signs the result.
func Seal(header any, nonceB64 string, plaintext, key []byte, priv ed25519.PrivateKey) ([]byte, error) {
	hdr, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if len(hdr) > MaxHeader {
		return nil, fmt.Errorf("segment: header of %d bytes exceeds the limit", len(hdr))
	}
	if len(plaintext) > MaxPlaintext {
		return nil, fmt.Errorf("segment: plaintext of %d bytes exceeds the limit", len(plaintext))
	}
	nonce, err := wire.DecodeB64(nonceB64)
	if err != nil || len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, errors.New("segment: invalid nonce")
	}
	var gz bytes.Buffer
	w, err := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	ct := aead.Seal(nil, nonce, gz.Bytes(), hdr)
	if len(ct) > MaxCiphertext {
		return nil, fmt.Errorf("segment: ciphertext of %d bytes exceeds the limit", len(ct))
	}
	out := make([]byte, 0, prefixLen+len(hdr)+len(ct)+sigLen)
	out = append(out, Magic...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(hdr)))
	out = append(out, hdr...)
	out = append(out, ct...)
	return append(out, ed25519.Sign(priv, out)...), nil
}

// EncodeOps returns the plaintext of a segment: the JSON array of ops.
func EncodeOps(ops []op.Op) ([]byte, error) { return json.Marshal(ops) }

// DecodeOps parses the plaintext of a segment strictly (§3): invalid UTF-8,
// duplicate keys, unknown fields and trailing data are rejected.
func DecodeOps(plain []byte) ([]op.Op, error) {
	var ops []op.Op
	if err := wire.DecodeStrict(plain, &ops); err != nil {
		return nil, fmt.Errorf("%w: ops: %v", ErrMalformed, err)
	}
	for i := range ops {
		for k, v := range ops[i].PK {
			ops[i].PK[k] = op.Normalize(v)
		}
		for k, v := range ops[i].Set {
			ops[i].Set[k] = op.Normalize(v)
		}
	}
	return ops, nil
}

// SchemaVersion is this app's sync schema version (§16). It is written into
// every segment and snapshot header; files with a higher version wait until
// the app is updated. Every migration that changes a synced table must raise
// it and add an upcaster (package engine).
const SchemaVersion = 1
