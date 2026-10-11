# ADR 0001: Collaboration and sync architecture

- **Status:** Accepted
- **Date:** 2026-10-11

## Context

Open ClubManager must start with zero configuration and support two modes:

1. **Local:** one person manages the club on one computer.
2. **Collaboration:** several club officials (board, treasurer, secretary,
   coaches) work on the same data. They are **not on the same network**, and
   the club has no one with DevOps skills.

The original plan for collaboration was to put the SQLite database file on a
shared drive. We rejected it:

- SQLite's file locking is unreliable on network file systems (SMB/NFS), and
  WAL mode does not work there at all. Concurrent writers risk corrupting the
  database.
- In practice clubs "share" via cloud folders (Nextcloud, OneDrive, Dropbox,
  Google Drive). These have no locking at all, produce conflict copies and may
  sync half-written files.
- Anyone with access to the share can read the file directly, including bank
  details (IBAN, SEPA mandates). App-level login, roles and audit log are
  bypassed.

The MySQL mode we had planned was also rejected, because running a database
server requires the DevOps skills we want to avoid.

Club data includes personal and banking data (GDPR). Access restrictions,
e.g. a coach must not see bank details, must be **enforced technically**,
not only hidden in the UI.

## Decision

### 1. Local-first, one SQLite per device

Every device keeps its own SQLite database, accessed by the Go backend
(pure-Go driver `modernc.org/sqlite`, no CGO). Migrations are embedded in the
Go binary and run on startup. There is no Prisma/Node at runtime.

In desktop mode the Go sidecar listens on `127.0.0.1` only.

### 2. Sync through immutable change files

Devices do not share a database. They exchange **change operations**:

```
/clubmanager-sync/
   club.json                              # club id, schema version, ACL (admin-signed)
   keys/<scope>/<epoch>/<device-id>.age   # scope key wrapped for each device
   devices/<device-id>/000001.ops.enc     # written only by this device, never modified
   devices/<device-id>/000002.ops.enc
   snapshots/<timestamp>.snap.enc         # periodic compaction
```

- Each file is written by exactly one device and never modified. This avoids
  write conflicts and makes cloud-folder sync safe.
- Every operation carries a **Hybrid Logical Clock** timestamp, the entity
  UUID, the **scope**, the schema version and the author device's signature.
- Conflict resolution is **last-writer-wins per field**. Deletes are
  tombstones. Append-only data (finance transactions, audit log) does not
  conflict.
- Primary keys are UUIDs. Unique business values (e.g. membership numbers) get
  explicit duplicate detection and resolution in the UI.
- A client refuses operations with a newer schema version and asks the user to
  update.

We do not adopt a generic CRDT library. The data model is small enough for
field-level LWW.

### 3. Pluggable transport

The sync engine talks to a minimal interface: `list`, `put`, `get` on
immutable files. Implementations, in order of delivery:

| Transport            | Use case                                                   | DevOps |
|----------------------|------------------------------------------------------------|--------|
| `LocalFolderTransport` | Folder synced by a Nextcloud/OneDrive/Dropbox client     | none   |
| `WebDAVTransport`    | Nextcloud directly via URL + app password                  | none   |
| `ServerTransport`    | Same Go binary in `--serve` mode (VPS, YunoHost, Pi, ...)  | low    |

All transports share the same protocol, so a club can move from a cloud folder
to a server without migrating data. In server mode the Go binary also serves
the web frontend (`go:embed`), so users can work in a browser without
installing anything.

### 4. Permissions by scope

Data is partitioned into scopes. Sensitive fields live in separate tables so
that each table belongs to exactly one scope.

| Scope             | Content                              | Board | Treasurer | Secretary | Coach |
|-------------------|--------------------------------------|-------|-----------|-----------|-------|
| `members.core`    | name, group, status, entry/exit      | rw    | r         | rw        | r     |
| `members.contact` | address, phone, e-mail               | rw    | r         | rw        | –     |
| `members.bank`    | IBAN, SEPA mandate                   | rw    | rw        | –         | –     |
| `finance`         | transactions, fees                   | rw    | rw        | –         | –     |
| `admin`           | roles, devices, settings             | rw    | –         | –         | –     |

Roles map to scopes and the mapping can be configured. The table shows the
defaults.

**Read access**
- Each scope has its own symmetric key (XChaCha20-Poly1305).
- Scope keys are wrapped for each authorized device's public key with `age`
  (X25519 recipients).
- A device without the key for a scope cannot decrypt that scope's data, even
  with full access to the shared folder.
- In server mode, the server additionally serves only the scopes a device may
  read.

**Write access**
- Each device signs its operations with its Ed25519 key.
- The admin maintains a signed ACL in `club.json` that maps device → role →
  scope → `r`/`rw`.
- Every client verifies signature and permission before applying an
  operation. Unauthorized operations are ignored.
- In server mode they are also rejected on upload.

**Group-level visibility** (e.g. a coach sees only their group) is available
in server mode only, at first. In folder mode it would require per-group keys
and complex re-keying when members change groups.

### 5. Identity and key lifecycle

- **Identity is a device key pair.** A local password or OS keychain protects
  the private key. JWT/password login applies only to browser access in server
  mode.
- **Invitation:**
  1. The admin creates an invite code or QR code.
  2. The new device generates its key pair and submits its public key.
  3. The admin approves the device, and the scope keys are wrapped for it.
- **Revocation:**
  1. Remove the device from the ACL.
  2. Create a new **key epoch** for every scope it could read.
  3. New data uses the new keys. Data the device already saw cannot be
     un-seen, in any architecture.
- **Recovery:** when a club is created, the app generates a recovery key and
  asks the user to print it and store it safely. Without it, losing the admin
  device means losing admin control.
- **At rest:** the local SQLite holds decrypted data. SQLCipher would require
  CGO, so we rely on OS disk encryption and recommend it during setup.

## Consequences

**Positive**
- Collaboration works without a server, using storage the club already has.
- Offline work is the normal case, not an edge case.
- The cloud storage provider only ever sees encrypted data.
- Permissions are enforced cryptographically, plus by the server in server
  mode.
- One protocol for all transports, so clubs can upgrade without migrating.
- The audit log is tamper-evident, because every operation is signed.

**Negative / risks**
- We own a sync protocol. That means more code than "SQLite on a share", and
  it needs good tests (property-based tests for convergence).
- Custom protocol around standard primitives. Use only vetted libraries
  (`age`, `crypto/ed25519`, `x/crypto/chacha20poly1305`). **The protocol must
  be reviewed by someone with cryptography experience before production use
  with real bank data.**
- Anyone with write access to a shared folder can delete files (an availability
  attack). Mitigation: full local copies on every device, snapshots and
  backups. Forged data is never applied.
- Key management (invites, revocation, recovery) adds UX the club must
  understand. It must be guided step by step in the app.
- Schema changes must stay compatible with old change files, and clubs need an
  update policy.

## Implementation order

1. **Data layer foundation:**
   - remove Prisma and MySQL
   - embedded Go migrations with `modernc.org/sqlite`
   - UUID primary keys
   - split tables by scope (e.g. `member_bank_details`)
   - `updated_at` / `deleted_at` columns
2. **Identity:** device keys, signed ACL, invite flow, recovery key.
3. **Sync core:** operation format, HLC, applying operations to SQLite,
   snapshots.
4. **Transports:** `LocalFolderTransport`, then `WebDAVTransport`, then
   `ServerTransport` (including group-level permissions and the embedded web
   frontend).

## Alternatives considered

| Alternative                                                   | Why rejected                                                              |
|---------------------------------------------------------------|---------------------------------------------------------------------------|
| SQLite file on a network share or cloud folder                | Corruption risk, no WAL, no access control (see Context)                  |
| MySQL server                                                  | Requires DevOps skills                                                    |
| One desktop app acts as LAN host                              | Officials are not on the same network; the host must always be online     |
| LAN host + tunnel (Tailscale, Cloudflare Tunnel)              | Every user needs extra setup or accounts; host must be online             |
| Hosted DB service (Turso/libSQL, Supabase)                    | Vendor dependency, GDPR data processing agreement for bank data, not simpler than our own server |
| Generic CRDT library (e.g. cr-sqlite, Automerge)              | Extra dependency (CGO/extension) for a model where field-level LWW is enough |
