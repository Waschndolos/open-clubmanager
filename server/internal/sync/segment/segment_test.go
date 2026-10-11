package segment

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/op"
)

const device = "abcdefghijklmnopqrstuvwxyz"

func fixture(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey, []byte, Header) {
	t.Helper()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	key := make([]byte, 32)
	rand.Read(key)
	nonce, err := NewNonce()
	if err != nil {
		t.Fatal(err)
	}
	h := Header{ClubID: "c", DeviceID: device, Scope: db.ScopeFinance, Seq: 1, Epoch: 1, ACLVersion: 1,
		Schema: 1, Nonce: nonce, HLCMin: "0000019a1b2c-0001", HLCMax: "0000019a1b2c-0001"}
	return pub, priv, key, h
}

func TestRoundTrip(t *testing.T) {
	pub, priv, key, h := fixture(t)
	ops := []op.Op{{T: "finance_transactions", PK: map[string]any{"id": "x"}, HLC: h.HLCMin, Set: map[string]any{"amount_cents": int64(1250)}}}
	plain, _ := EncodeOps(ops)
	raw, err := Seal(h, h.Nonce, plain, key, priv)
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Verify(pub); err != nil {
		t.Fatal(err)
	}
	got, err := f.SegmentHeader()
	if err != nil || *got != h {
		t.Fatalf("header %+v, %v", got, err)
	}
	out, err := f.Open(key, got.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecodeOps(out)
	if err != nil || len(dec) != 1 || dec[0].Set["amount_cents"] != int64(1250) {
		t.Fatalf("ops %+v, %v", dec, err)
	}

	// Every bit of the file is covered: header (AAD and signature),
	// ciphertext (AEAD tag and signature), signature.
	for _, i := range []int{0, 5, 10, len(raw) - 70, len(raw) - 1} {
		bad := bytes.Clone(raw)
		bad[i] ^= 1
		f, err := Parse(bad)
		if err == nil {
			err = f.Verify(pub)
		}
		if err == nil {
			t.Errorf("flipped byte %d accepted", i)
		}
	}
	// The header is bound to the ciphertext even without the signature.
	f2, _ := Parse(raw)
	f2.Header = bytes.Replace(bytes.Clone(f2.Header), []byte(`"seq":1`), []byte(`"seq":2`), 1)
	if _, err := f2.Open(key, h.Nonce); !errors.Is(err, ErrDecrypt) {
		t.Errorf("changed header decrypts: %v", err)
	}
	// Nonces are fresh.
	if n2, _ := NewNonce(); n2 == h.Nonce {
		t.Error("nonce repeated")
	}
}

func TestLimitsAndFraming(t *testing.T) {
	_, priv, key, h := fixture(t)
	raw, _ := Seal(h, h.Nonce, []byte("[]"), key, priv)
	for name, in := range map[string][]byte{
		"empty":       nil,
		"bad magic":   append([]byte("OCM2"), raw[4:]...),
		"truncated":   raw[:len(raw)/2],
		"header 0":    append(append([]byte("OCM1"), 0, 0, 0, 0), raw[8:]...),
		"header huge": append(append([]byte("OCM1"), binary.BigEndian.AppendUint32(nil, MaxHeader+1)...), raw[8:]...),
	} {
		if _, err := Parse(in); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v", name, err)
		}
	}
	big := make([]byte, 8+10+MaxCiphertext+1+64)
	copy(big, "OCM1")
	binary.BigEndian.PutUint32(big[4:], 10)
	if _, err := Parse(big); !errors.Is(err, ErrMalformed) {
		t.Errorf("oversized ciphertext: %v", err)
	}
}

func TestHeaderStrict(t *testing.T) {
	for name, hdr := range map[string]string{
		"unknown field":  `{"club_id":"c","device_id":"` + device + `","scope":"finance","seq":1,"prev":null,"epoch":1,"acl_version":1,"schema":1,"nonce":"x","hlc_min":"0000019a1b2c-0001","hlc_max":"0000019a1b2c-0001","extra":1}`,
		"duplicate key":  `{"seq":1,"seq":2}`,
		"prev on seq 1":  `{"club_id":"c","device_id":"` + device + `","scope":"finance","seq":1,"prev":"sha256:AAAA","epoch":1,"acl_version":1,"schema":1,"nonce":"x","hlc_min":"0000019a1b2c-0001","hlc_max":"0000019a1b2c-0001"}`,
		"unknown scope":  `{"club_id":"c","device_id":"` + device + `","scope":"secret","seq":1,"prev":null,"epoch":1,"acl_version":1,"schema":1,"nonce":"x","hlc_min":"0000019a1b2c-0001","hlc_max":"0000019a1b2c-0001"}`,
		"inverted range": `{"club_id":"c","device_id":"` + device + `","scope":"finance","seq":1,"prev":null,"epoch":1,"acl_version":1,"schema":1,"nonce":"x","hlc_min":"0000019a1b2c-0002","hlc_max":"0000019a1b2c-0001"}`,
	} {
		f := &Frame{Header: []byte(hdr)}
		if _, err := f.SegmentHeader(); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDecompressionLimits(t *testing.T) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(strings.Repeat("a", MaxPlaintext+1)))
	w.Close()
	if _, err := gunzip(buf.Bytes()); !errors.Is(err, ErrMalformed) {
		t.Errorf("zip bomb: %v", err)
	}
	buf.Reset()
	w = gzip.NewWriter(&buf)
	w.Write([]byte("[]"))
	w.Close()
	if _, err := gunzip(append(buf.Bytes(), "trailer"...)); !errors.Is(err, ErrMalformed) {
		t.Errorf("trailing data: %v", err)
	}
	if out, err := gunzip(buf.Bytes()); err != nil || string(out) != "[]" {
		t.Errorf("plain: %q %v", out, err)
	}
}
