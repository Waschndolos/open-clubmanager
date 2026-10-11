package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Ensure returns this installation's device identity, creating it on first
// use (§5.1). On creation it stores the private keys with ks, records the
// device id and key references in sync_device and, in the same transaction,
// assigns the new device id to field clocks written before the identity
// existed (device_id NULL meant "this device", see migration 00002).
func Ensure(ctx context.Context, conn *sql.DB, ks KeyStore) (*Device, error) {
	d, err := Load(ctx, conn, ks)
	if err == nil || !errors.Is(err, ErrNoIdentity) {
		return d, err
	}

	k, err := Generate()
	if err != nil {
		return nil, err
	}
	d = NewDevice(k)
	ref, err := ks.Save(d)
	if err != nil {
		return nil, fmt.Errorf("identity: store device keys: %w", err)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE sync_device SET device_id = ?, sign_key_ref = ?, enc_key_ref = ?, sign_pub = ?, age_recipient = ?
		 WHERE id = 1 AND device_id IS NULL`,
		d.ID, ref, ref, d.SignPubB64(), d.AgeRecipient())
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, errors.New("identity: device identity was created concurrently")
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE sync_field_clock SET device_id = ? WHERE device_id IS NULL`, d.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

// ErrNoIdentity is returned by Load before the device identity exists.
var ErrNoIdentity = errors.New("identity: device identity not created yet")

// Load returns the existing device identity.
func Load(ctx context.Context, conn *sql.DB, ks KeyStore) (*Device, error) {
	var id, ref sql.NullString
	if err := conn.QueryRowContext(ctx,
		`SELECT device_id, sign_key_ref FROM sync_device WHERE id = 1`).Scan(&id, &ref); err != nil {
		return nil, err
	}
	if !id.Valid {
		return nil, ErrNoIdentity
	}
	k, err := ks.Load(ref.String)
	if err != nil {
		return nil, err
	}
	d := NewDevice(k)
	if d.ID != id.String {
		return nil, fmt.Errorf("identity: stored keys belong to %s, not %s", d.ID, id.String)
	}
	return d, nil
}
