package acl

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"filippo.io/age"
	"github.com/google/uuid"

	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// ProtocolVersion is the sync protocol version this package implements.
const ProtocolVersion = 1

const genesisType = "ocm.club"

// Genesis is club.json (§7).
type Genesis struct {
	Type             string `json:"type"`
	Protocol         int    `json:"protocol"`
	ClubID           string `json:"club_id"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
	RootSignPub      string `json:"root_sign_pub"`
	RootAgeRecipient string `json:"root_age_recipient"`

	rootPub ed25519.PublicKey
}

// ErrBadGenesis is returned for a club.json that is malformed or not signed
// by its own root key.
var ErrBadGenesis = errors.New("acl: invalid club.json")

// ErrGenesisMismatch is returned when club.json differs from the pinned
// club_id or root key (§7).
var ErrGenesisMismatch = errors.New("acl: club.json does not match the pinned club")

// NewGenesis builds and signs club.json with the root key.
func NewGenesis(clubID, name string, createdAt time.Time, root ed25519.PrivateKey, rootAge string) (raw, sig []byte, err error) {
	g := Genesis{
		Type:             genesisType,
		Protocol:         ProtocolVersion,
		ClubID:           clubID,
		Name:             name,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		RootSignPub:      wire.EncodeB64(root.Public().(ed25519.PublicKey)),
		RootAgeRecipient: rootAge,
	}
	raw, err = json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return raw, wire.Sign(root, raw), nil
}

// ParseGenesis parses club.json and verifies its root signature. Pinning is
// up to the caller (see Genesis.Matches).
func ParseGenesis(raw, sig []byte) (*Genesis, error) {
	var g Genesis
	if err := wire.DecodeStrict(raw, &g); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadGenesis, err)
	}
	if g.Type != genesisType {
		return nil, fmt.Errorf("%w: type %q", ErrBadGenesis, g.Type)
	}
	if g.Protocol != ProtocolVersion {
		return nil, fmt.Errorf("%w: unsupported protocol %d", ErrBadGenesis, g.Protocol)
	}
	if _, err := uuid.Parse(g.ClubID); err != nil {
		return nil, fmt.Errorf("%w: club_id: %v", ErrBadGenesis, err)
	}
	pub, err := wire.ParsePublicKey(g.RootSignPub)
	if err != nil {
		return nil, fmt.Errorf("%w: root_sign_pub: %v", ErrBadGenesis, err)
	}
	if _, err := age.ParseX25519Recipient(g.RootAgeRecipient); err != nil {
		return nil, fmt.Errorf("%w: root_age_recipient: %v", ErrBadGenesis, err)
	}
	if err := wire.Verify(pub, raw, sig); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadGenesis, err)
	}
	g.rootPub = pub
	return &g, nil
}

// RootPub returns the root signing key.
func (g *Genesis) RootPub() ed25519.PublicKey { return g.rootPub }

// Matches checks g against the pinned club_id and root_sign_pub (§7).
func (g *Genesis) Matches(clubID, rootSignPub string) error {
	if g.ClubID != clubID || g.RootSignPub != rootSignPub {
		return ErrGenesisMismatch
	}
	return nil
}
