package wire

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
)

func TestB64(t *testing.T) {
	b := []byte{0xfb, 0xff, 0x01}
	s := EncodeB64(b)
	if s != "-_8B" {
		t.Fatalf("EncodeB64 = %q", s)
	}
	for _, bad := range []string{"+/8B", "-_8B==", "-_9", "-_8B!"} {
		if _, err := DecodeB64(bad); err == nil {
			t.Errorf("DecodeB64(%q) accepted", bad)
		}
	}
}

func TestHash(t *testing.T) {
	h := Hash([]byte("abc"))
	if h != "sha256:ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0" {
		t.Fatalf("Hash = %s", h)
	}
	sum, err := ParseHash(h)
	if err != nil || FormatHash(sum) != h {
		t.Fatalf("ParseHash: %v", err)
	}
	for _, bad := range []string{"sha1:abc", "sha256:", "sha256:AAAA", "ungWv48Bz-pBQUDeXa4iI7ADYaOWF3qctBD_YfIAFa0"} {
		if _, err := ParseHash(bad); err == nil {
			t.Errorf("ParseHash(%q) accepted", bad)
		}
	}
}

func TestSignVerify(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	data := []byte(`{"a":1}`)
	sig := Sign(priv, data)
	if err := Verify(pub, data, sig); err != nil {
		t.Fatal(err)
	}
	if err := Verify(pub, data, append(sig, '\n')); err != nil {
		t.Errorf("trailing newline: %v", err)
	}
	for name, err := range map[string]error{
		"other key":     Verify(other, data, sig),
		"other data":    Verify(pub, []byte(`{"a":2}`), sig),
		"empty sig":     Verify(pub, data, nil),
		"garbage sig":   Verify(pub, data, []byte("not base64!")),
		"short pub key": Verify(pub[:31], data, sig),
	} {
		if !errors.Is(err, ErrBadSignature) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestDecodeStrict(t *testing.T) {
	var v struct {
		A int `json:"a"`
	}
	if err := DecodeStrict([]byte(` {"a": 1} `), &v); err != nil || v.A != 1 {
		t.Fatalf("valid: %v", err)
	}
	for _, bad := range []string{`{"a":1,"b":2}`, `{"a":1}{"a":2}`, `{"a":1} x`, `{"a":"1"}`, ``} {
		if err := DecodeStrict([]byte(bad), &v); err == nil {
			t.Errorf("DecodeStrict(%q) accepted", bad)
		}
	}
}

func TestDecodeStrictRejectsDuplicatesAndBadUTF8(t *testing.T) {
	var v map[string]any
	for _, in := range []string{
		`{"a": 1, "a": 2}`,
		`{"x": {"b": [1, {"c": 1, "c": 2}]}}`,
		"{\"a\": \"\xff\"}",
	} {
		if err := DecodeStrict([]byte(in), &v); err == nil {
			t.Errorf("accepted %q", in)
		}
	}
	for _, in := range []string{`{"a": {"a": 1}, "b": [{"a": 1}, {"a": 2}]}`, `{"a": [], "b": {}}`} {
		if err := DecodeStrict([]byte(in), &v); err != nil {
			t.Errorf("rejected %q: %v", in, err)
		}
	}
}
