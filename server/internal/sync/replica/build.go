package replica

import (
	"context"
	"crypto/ed25519"
	"database/sql"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/hlc"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/layout"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/segment"
	"github.com/Waschndolos/open-clubmanager/server/internal/sync/wire"
)

// SnapshotSpec names everything a snapshot file needs besides the state.
type SnapshotSpec struct {
	ClubID     string
	DeviceID   string
	Sign       ed25519.PrivateKey
	Scope      db.Scope
	Epoch      uint64
	Key        []byte
	ACLVersion uint64
	HLC        hlc.Timestamp
	Streams    map[string]segment.Frontier
}

// WriteSnapshot reads the state of a scope in tx, seals it into a snapshot
// file (§15.1), queues the file for upload and records it as the scope's
// base on this device. It returns the path and the complete-file hash.
func WriteSnapshot(ctx context.Context, tx *sql.Tx, spec SnapshotSpec) (string, string, error) {
	state, err := ReadState(ctx, tx, spec.Scope)
	if err != nil {
		return "", "", err
	}
	plain, err := state.Encode()
	if err != nil {
		return "", "", err
	}
	nonce, err := segment.NewNonce()
	if err != nil {
		return "", "", err
	}
	streams := spec.Streams
	if streams == nil {
		streams = map[string]segment.Frontier{}
	}
	hdr := segment.SnapshotHeader{
		Type: segment.SnapshotType, ClubID: spec.ClubID, DeviceID: spec.DeviceID, Scope: spec.Scope,
		Streams: streams, Epoch: spec.Epoch, ACLVersion: spec.ACLVersion, Schema: segment.SchemaVersion,
		Nonce: nonce, HLC: spec.HLC.String(),
	}
	data, err := segment.Seal(hdr, nonce, plain, spec.Key, spec.Sign)
	if err != nil {
		return "", "", err
	}
	path := layout.SnapshotPath(spec.Scope, spec.Epoch, hdr.HLC, spec.DeviceID)
	hash := wire.Hash(data)
	if err := QueueUpload(ctx, tx, path, data); err != nil {
		return "", "", err
	}
	if err := SetBase(ctx, tx, spec.Scope, path, hash); err != nil {
		return "", "", err
	}
	return path, hash, nil
}

// SetBase records the snapshot a scope's local state started from.
func SetBase(ctx context.Context, tx *sql.Tx, scope db.Scope, path, hash string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO sync_scopes (scope, snapshot, hash) VALUES (?, ?, ?)
		 ON CONFLICT (scope) DO UPDATE SET snapshot = excluded.snapshot, hash = excluded.hash,
		   loaded_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')`, scope, path, hash)
	return err
}
