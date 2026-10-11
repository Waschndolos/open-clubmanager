package identity

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// ErrNoKeys is returned by KeyStore.Load when no keys are stored under ref.
var ErrNoKeys = errors.New("identity: no stored device keys")

// ErrWrongPassphrase is returned when the stored keys cannot be decrypted.
var ErrWrongPassphrase = errors.New("identity: wrong passphrase")

// KeyStore keeps the private device keys encrypted at rest (§5.1). The OS
// keychain will be a second implementation; the passphrase file is the
// fallback the spec allows.
type KeyStore interface {
	// Save stores the keys of a device and returns a reference to them, which
	// is kept in sync_device.
	Save(d *Device) (ref string, err error)
	// Load returns the keys stored under ref.
	Load(ref string) (*KeyPair, error)
}

// fileRefPrefix marks key references of PassphraseFileStore.
const fileRefPrefix = "file:"

// PassphraseFileStore stores the keys in "<Dir>/device-<id>.key.age",
// encrypted with an age scrypt recipient for the user's local passphrase.
type PassphraseFileStore struct {
	Dir        string
	Passphrase string
	// WorkFactor is the scrypt log2(N). Zero means age's default (18), which
	// takes about a second; tests use a small value.
	WorkFactor int
}

// storedKeys is the plaintext inside the encrypted file.
type storedKeys struct {
	Type        string `json:"type"`
	DeviceID    string `json:"device_id"`
	SignSeed    string `json:"sign_seed"`
	AgeIdentity string `json:"age_identity"`
}

const storedKeysType = "ocm.device-keys"

func (s *PassphraseFileStore) Save(d *Device) (string, error) {
	if s.Passphrase == "" {
		return "", errors.New("identity: empty passphrase")
	}
	r, err := age.NewScryptRecipient(s.Passphrase)
	if err != nil {
		return "", err
	}
	if s.WorkFactor > 0 {
		r.SetWorkFactor(s.WorkFactor)
	}
	plain, err := json.Marshal(storedKeys{
		Type:        storedKeysType,
		DeviceID:    d.ID,
		SignSeed:    wire.EncodeB64(d.Sign.Seed()),
		AgeIdentity: d.Age.String(),
	})
	if err != nil {
		return "", err
	}
	defer clear(plain)

	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, r)
	if err != nil {
		return "", err
	}
	if _, err := w.Write(plain); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	name := "device-" + d.ID + ".key.age"
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return "", err
	}
	if err := writeFileAtomic(filepath.Join(s.Dir, name), buf.Bytes()); err != nil {
		return "", err
	}
	return fileRefPrefix + name, nil
}

func (s *PassphraseFileStore) Load(ref string) (*KeyPair, error) {
	name, ok := strings.CutPrefix(ref, fileRefPrefix)
	if !ok || name != filepath.Base(name) || name == "." || name == ".." {
		return nil, fmt.Errorf("identity: invalid key reference %q", ref)
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNoKeys
	}
	if err != nil {
		return nil, err
	}
	id, err := age.NewScryptIdentity(s.Passphrase)
	if err != nil {
		return nil, err
	}
	rd, err := age.Decrypt(bytes.NewReader(data), id)
	if err != nil {
		var nm *age.NoIdentityMatchError
		if errors.As(err, &nm) {
			return nil, ErrWrongPassphrase
		}
		return nil, fmt.Errorf("identity: decrypt key file: %w", err)
	}
	plain, err := io.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("identity: decrypt key file: %w", err)
	}
	defer clear(plain)

	var sk storedKeys
	if err := wire.DecodeStrict(plain, &sk); err != nil || sk.Type != storedKeysType {
		return nil, errors.New("identity: malformed key file")
	}
	seed, err := wire.DecodeB64(sk.SignSeed)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("identity: malformed key file")
	}
	defer clear(seed)
	ageID, err := age.ParseX25519Identity(sk.AgeIdentity)
	if err != nil {
		return nil, errors.New("identity: malformed key file")
	}
	k := &KeyPair{Sign: ed25519.NewKeyFromSeed(seed), Age: ageID}
	if DeviceID(k.SignPub()) != sk.DeviceID {
		return nil, errors.New("identity: key file does not match its device id")
	}
	return k, nil
}

// writeFileAtomic writes data to a temp file next to path and renames it.
func writeFileAtomic(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-key-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
