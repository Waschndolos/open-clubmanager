package identity

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

func TestDeviceIDVector(t *testing.T) {
	seed := make([]byte, 32)
	for i := range seed {
		seed[i] = byte(i)
	}
	pub := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
	if hex.EncodeToString(pub) != "03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8" {
		t.Fatalf("unexpected public key %x", pub)
	}
	// Computed independently: base32(sha256(pub)[:16]).lower() without "=".
	if got := DeviceID(pub); got != "kzdvvj2umnduyauf35o36k6kw4" {
		t.Errorf("DeviceID = %s", got)
	}
	// SHA-256(pub || "age1test")[:8] as big-endian integer mod 10^6.
	if got := SafetyCode(pub, "age1test"); got != "994998" {
		t.Errorf("SafetyCode = %s", got)
	}
}

func TestDeviceIDFormat(t *testing.T) {
	re := regexp.MustCompile(`^[a-z2-7]{26}$`)
	for range 50 {
		k, err := Generate()
		if err != nil {
			t.Fatal(err)
		}
		if id := DeviceID(k.SignPub()); !re.MatchString(id) {
			t.Fatalf("device id %q", id)
		}
		if c := SafetyCode(k.SignPub(), k.AgeRecipient()); len(c) != 6 || strings.Trim(c, "0123456789") != "" {
			t.Fatalf("safety code %q", c)
		}
	}
}

func testStore(t *testing.T) *PassphraseFileStore {
	return &PassphraseFileStore{Dir: t.TempDir(), Passphrase: "correct horse", WorkFactor: 10}
}

func TestKeyStoreRoundTrip(t *testing.T) {
	ks := testStore(t)
	k, _ := Generate()
	d := NewDevice(k)
	ref, err := ks.Save(d)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ks.Load(ref)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Sign, d.Sign) || got.Age.String() != d.Age.String() {
		t.Fatal("loaded keys differ")
	}

	// The file must not contain the private keys in the clear.
	data, err := os.ReadFile(filepath.Join(ks.Dir, strings.TrimPrefix(ref, "file:")))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{wire.EncodeB64(d.Sign.Seed()), d.Age.String(), string(d.Sign.Seed())} {
		if bytes.Contains(data, []byte(secret)) {
			t.Fatal("key file contains a private key in the clear")
		}
	}

	wrong := &PassphraseFileStore{Dir: ks.Dir, Passphrase: "wrong"}
	if _, err := wrong.Load(ref); !errors.Is(err, ErrWrongPassphrase) {
		t.Errorf("wrong passphrase: %v", err)
	}
	if _, err := ks.Load("file:missing.key.age"); !errors.Is(err, ErrNoKeys) {
		t.Errorf("missing: %v", err)
	}
	for _, bad := range []string{"keychain:x", "file:../x", "file:a/b", "file:.."} {
		if _, err := ks.Load(bad); err == nil {
			t.Errorf("ref %q accepted", bad)
		}
	}
	if _, err := (&PassphraseFileStore{Dir: ks.Dir}).Save(d); err == nil {
		t.Error("empty passphrase accepted")
	}
}

func TestRecoveryKit(t *testing.T) {
	root, _ := Generate()
	kit := &RecoveryKit{ClubID: "0190f5c2-7b1e-7c3a-9d2e-4b8a1f6e3c21", ClubName: "TSV", Root: root}
	text := kit.Text()
	got, err := ParseRecoveryKit(text)
	if err != nil {
		t.Fatal(err)
	}
	if got.ClubID != kit.ClubID || !bytes.Equal(got.Root.Sign, root.Sign) || got.Root.AgeRecipient() != root.AgeRecipient() {
		t.Fatal("parsed kit differs")
	}
	// Only the key lines are needed, with surrounding white space.
	var keyLines []string
	for _, l := range strings.Split(text, "\n") {
		if strings.Contains(l, ": ") && (strings.HasPrefix(l, "root-") || strings.HasPrefix(l, "check") || strings.HasPrefix(l, "club-id")) {
			keyLines = append(keyLines, "  "+l+"  ")
		}
	}
	if _, err := ParseRecoveryKit(strings.Join(keyLines, "\r\n")); err != nil {
		t.Errorf("key lines only: %v", err)
	}

	seedB64 := wire.EncodeB64(root.Sign.Seed())
	typo := []byte(seedB64)
	if typo[5] == 'A' {
		typo[5] = 'B'
	} else {
		typo[5] = 'A'
	}
	for name, bad := range map[string]string{
		"typo in signing key": strings.Replace(text, seedB64, string(typo), 1),
		"other club":          strings.Replace(text, kit.ClubID, "0190f5c2-7b1e-7c3a-9d2e-4b8a1f6e3c22", 1),
		"missing age key":     strings.Replace(text, kitAge, "x", 1),
		"empty":               "",
	} {
		if _, err := ParseRecoveryKit(bad); !errors.Is(err, ErrBadRecoveryKit) {
			t.Errorf("%s: %v", name, err)
		}
	}

	kit.Wipe()
	if kit.Root != nil {
		t.Error("Wipe kept the root key")
	}
	k2, _ := Generate()
	sign := k2.Sign
	k2.Wipe()
	if !bytes.Equal(sign, make([]byte, len(sign))) {
		t.Error("Wipe did not clear the signing key")
	}
}

func TestEnsureBackfillsFieldClocks(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(ctx, filepath.Join(t.TempDir(), "club.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Field clocks written before the identity exists carry NULL.
	if _, err := conn.Exec(`INSERT INTO sync_field_clock (tbl, pk, col, hlc, device_id) VALUES
		('members', '{"id":"a"}', 'first_name', '000000000001-0000', NULL),
		('members', '{"id":"b"}', 'first_name', '000000000001-0000', 'otherdeviceaaaaaaaaaaaaaaa')`); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(ctx, conn, testStore(t)); !errors.Is(err, ErrNoIdentity) {
		t.Fatalf("Load before Ensure: %v", err)
	}

	ks := testStore(t)
	d, err := Ensure(ctx, conn, ks)
	if err != nil {
		t.Fatal(err)
	}
	var id, signPub, ageRcpt, ref string
	if err := conn.QueryRow(`SELECT device_id, sign_pub, age_recipient, sign_key_ref FROM sync_device`).Scan(&id, &signPub, &ageRcpt, &ref); err != nil {
		t.Fatal(err)
	}
	if id != d.ID || signPub != d.SignPubB64() || ageRcpt != d.AgeRecipient() || !strings.HasPrefix(ref, "file:") {
		t.Errorf("sync_device = %s %s %s %s", id, signPub, ageRcpt, ref)
	}
	var nulls, own, other int
	conn.QueryRow(`SELECT COUNT(*) FILTER (WHERE device_id IS NULL), COUNT(*) FILTER (WHERE device_id = ?),
		COUNT(*) FILTER (WHERE device_id = 'otherdeviceaaaaaaaaaaaaaaa') FROM sync_field_clock`, d.ID).Scan(&nulls, &own, &other)
	if nulls != 0 || own != 1 || other != 1 {
		t.Errorf("field clocks: %d NULL, %d own, %d other", nulls, own, other)
	}

	again, err := Ensure(ctx, conn, ks)
	if err != nil || again.ID != d.ID {
		t.Fatalf("second Ensure: %v, %v", again, err)
	}
	if _, err := Load(ctx, conn, &PassphraseFileStore{Dir: ks.Dir, Passphrase: "nope"}); !errors.Is(err, ErrWrongPassphrase) {
		t.Errorf("Load with wrong passphrase: %v", err)
	}
}
