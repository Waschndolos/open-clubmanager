package keys

import (
	"bytes"
	"errors"
	"testing"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

func TestWrapUnwrap(t *testing.T) {
	k, err := New()
	if err != nil {
		t.Fatal(err)
	}
	id, _ := age.GenerateX25519Identity()
	other, _ := age.GenerateX25519Identity()

	w, err := Wrap(k, id.Recipient().String())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(w, k) {
		t.Fatal("wrapped key contains the key")
	}
	got, err := Unwrap(w, id)
	if err != nil || !bytes.Equal(got, k) {
		t.Fatalf("Unwrap: %v", err)
	}
	if _, err := Unwrap(w, other); err == nil {
		t.Error("unwrapped with another identity")
	}
	w2, _ := Wrap(k, id.Recipient().String())
	if bytes.Equal(w, w2) {
		t.Error("wrapping is deterministic")
	}
	if _, err := Wrap(k, "age1invalid"); err == nil {
		t.Error("invalid recipient accepted")
	}

	// A wrapped value that is not 32 bytes long is rejected.
	short, _ := Wrap(k[:16], id.Recipient().String())
	if _, err := Unwrap(short, id); err == nil {
		t.Error("short key accepted")
	}
}

func TestCheck(t *testing.T) {
	k, _ := New()
	c := Check(db.ScopeFinance, 2, k)
	if err := Verify(db.ScopeFinance, 2, k, c); err != nil {
		t.Fatal(err)
	}
	other, _ := New()
	for name, err := range map[string]error{
		"other key":   Verify(db.ScopeFinance, 2, other, c),
		"other epoch": Verify(db.ScopeFinance, 3, k, c),
		"other scope": Verify(db.ScopeMembersBank, 2, k, c),
		"short key":   Verify(db.ScopeFinance, 2, k[:31], c),
	} {
		if !errors.Is(err, ErrCheckMismatch) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
