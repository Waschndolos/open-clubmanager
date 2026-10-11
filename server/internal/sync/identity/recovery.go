package identity

import (
	"bufio"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"filippo.io/age"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// RecoveryKit holds the private parts of a club's root key (§5.2). It exists
// only in memory: it is shown once, the user confirms they printed or saved
// it, and then Wipe deletes it. Nothing in this package writes it to disk.
type RecoveryKit struct {
	ClubID    string
	ClubName  string
	CreatedAt time.Time
	Root      *KeyPair
}

// Wipe deletes the root private key from memory (§17.1 step 3, §17.5 step 3).
func (k *RecoveryKit) Wipe() {
	if k.Root != nil {
		k.Root.Wipe()
		k.Root = nil
	}
}

const (
	kitHeader  = "OPEN CLUBMANAGER RECOVERY KIT"
	kitSign    = "root-sign-key:"
	kitAge     = "root-age-key:"
	kitCheck   = "check:"
	kitClubID  = "club-id:"
	kitVersion = "kit-version: 1"
)

// kitChecksum lets ParseRecoveryKit detect typing errors in the signing key
// before it is compared with the genesis file. The age identity carries its
// own bech32 checksum.
func kitChecksum(clubID string, seed []byte, ageID string) string {
	h := sha256.New()
	h.Write([]byte(clubID + "\n"))
	h.Write(seed)
	h.Write([]byte("\n" + ageID))
	return hex.EncodeToString(h.Sum(nil)[:4])
}

// Text renders the kit for printing. The QR code is a later UI step; it will
// encode this same text.
func (k *RecoveryKit) Text() string {
	seed := k.Root.Sign.Seed()
	ageID := k.Root.Age.String()
	var b strings.Builder
	fmt.Fprintln(&b, kitHeader)
	fmt.Fprintln(&b, kitVersion)
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Verein / Club: %s\n", k.ClubName)
	fmt.Fprintf(&b, "Erstellt / Created: %s\n", k.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Dieses Blatt stellt die Verwaltung des Vereins wieder her, wenn kein")
	fmt.Fprintln(&b, "Admin-Gerät mehr verfügbar ist. Sicher und offline aufbewahren.")
	fmt.Fprintln(&b, "This sheet restores administration of the club if no admin device is")
	fmt.Fprintln(&b, "left. Keep it safe and offline. Anyone holding it controls the club.")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s %s\n", kitClubID, k.ClubID)
	fmt.Fprintf(&b, "%s %s\n", kitSign, wire.EncodeB64(seed))
	fmt.Fprintf(&b, "%s %s\n", kitAge, ageID)
	fmt.Fprintf(&b, "%s %s\n", kitCheck, kitChecksum(k.ClubID, seed, ageID))
	return b.String()
}

// ErrBadRecoveryKit is returned for kits that cannot be parsed.
var ErrBadRecoveryKit = errors.New("identity: malformed recovery kit")

// ParseRecoveryKit reads the key lines of a kit produced by Text. Other lines
// are ignored, so a kit typed in with or without the explanatory text works.
// The caller must compare the public keys with the pinned genesis file.
func ParseRecoveryKit(text string) (*RecoveryKit, error) {
	var clubID, signLine, ageLine, check string
	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, kitClubID):
			clubID = strings.TrimSpace(strings.TrimPrefix(line, kitClubID))
		case strings.HasPrefix(line, kitSign):
			signLine = strings.TrimSpace(strings.TrimPrefix(line, kitSign))
		case strings.HasPrefix(line, kitAge):
			ageLine = strings.TrimSpace(strings.TrimPrefix(line, kitAge))
		case strings.HasPrefix(line, kitCheck):
			check = strings.TrimSpace(strings.TrimPrefix(line, kitCheck))
		}
	}
	if clubID == "" || signLine == "" || ageLine == "" || check == "" {
		return nil, fmt.Errorf("%w: missing lines", ErrBadRecoveryKit)
	}
	seed, err := wire.DecodeB64(signLine)
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%w: root signing key", ErrBadRecoveryKit)
	}
	defer clear(seed)
	ageID, err := age.ParseX25519Identity(ageLine)
	if err != nil {
		return nil, fmt.Errorf("%w: root age key", ErrBadRecoveryKit)
	}
	if kitChecksum(clubID, seed, ageLine) != check {
		return nil, fmt.Errorf("%w: checksum mismatch (typing error?)", ErrBadRecoveryKit)
	}
	return &RecoveryKit{
		ClubID: clubID,
		Root:   &KeyPair{Sign: ed25519.NewKeyFromSeed(seed), Age: ageID},
	}, nil
}
