# Sync protocol specification

- **Status:** Draft. Needs review by someone with cryptography experience
  before implementation is used with real data.
- **Protocol version:** 1
- **Based on:** [ADR 0001](adr/0001-collaboration-and-sync-architecture.md)
- **Data model:** [data-model.md](data-model.md)

The key words MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.

## Contents

1. [Goals and non-goals](#1-goals-and-non-goals)
2. [Terms](#2-terms)
3. [Cryptographic primitives](#3-cryptographic-primitives)
4. [Scopes and roles](#4-scopes-and-roles)
5. [Identities and keys](#5-identities-and-keys)
6. [Storage layout](#6-storage-layout)
7. [Club genesis](#7-club-genesis)
8. [Access control list (ACL)](#8-access-control-list-acl)
9. [Scope keys and epochs](#9-scope-keys-and-epochs)
10. [Change operations](#10-change-operations)
11. [Segments](#11-segments)
12. [Hybrid logical clock](#12-hybrid-logical-clock)
13. [Applying changes](#13-applying-changes)
14. [Conflict rules](#14-conflict-rules)
15. [Snapshots, compaction and erasure](#15-snapshots-compaction-and-erasure)
16. [Schema evolution](#16-schema-evolution)
17. [Device lifecycle](#17-device-lifecycle)
18. [Transports](#18-transports)
19. [Server mode](#19-server-mode)
20. [Local state](#20-local-state)
21. [Security considerations](#21-security-considerations)
22. [Required changes to the current data layer](#22-required-changes-to-the-current-data-layer)
23. [Decisions and remaining limitations](#23-decisions-and-remaining-limitations)

---

## 1. Goals and non-goals

**Goals**
- Several devices that are not on the same network work on the same club data.
- No server is required. A folder synced by Nextcloud, OneDrive, Dropbox or
  Google Drive, or a WebDAV share, is enough.
- The storage provider and anyone with access to the shared folder but no
  keys learn nothing about the content.
- Read and write permissions are enforced per scope, by cryptography (read)
  and signature verification (write). They do not depend on honest clients.
- Every device converges to the same state once it has seen the same changes.
- Offline work is normal.
- Every change can be attributed to a device.

**Non-goals (version 1)**
- Real-time collaboration (seconds of latency are fine, minutes are acceptable).
- Hiding metadata: the number of devices, scope names, timing and size of
  changes are visible to the storage provider (see [§21](#21-security-considerations)).
- Per-group visibility in folder mode (only in server mode, see [§19](#19-server-mode)).
- Merging text edits within a field. Field values are replaced as a whole.

## 2. Terms

| Term | Meaning |
|---|---|
| **Club** | One data set, identified by `club_id`. |
| **Device** | An installation of the app with its own key pairs. A person can have several devices. |
| **Scope** | A named partition of data with its own key and its own permissions. |
| **Epoch** | A generation of a scope key. A new epoch starts on revocation. |
| **Operation (op)** | One change to one row, see [§10](#10-change-operations). |
| **Segment** | An immutable, encrypted and signed file with a batch of ops of one scope from one device. |
| **Stream** | The sequence of segments of one device in one scope. |
| **ACL** | The signed list of devices, roles and permissions. |
| **Root key** | The club's top-level key pair, held offline in the recovery kit. |
| **HLC** | Hybrid logical clock timestamp, see [§12](#12-hybrid-logical-clock). |

## 3. Cryptographic primitives

Implementations MUST use these primitives and MUST NOT implement them themselves.

| Purpose | Primitive | Go package |
|---|---|---|
| Signatures | Ed25519 | `crypto/ed25519` |
| Scope data encryption | XChaCha20-Poly1305, 32-byte key, 24-byte random nonce | `golang.org/x/crypto/chacha20poly1305` |
| Key wrapping for devices | age, X25519 recipients | `filippo.io/age` |
| Invite proof | HMAC-SHA-256 | `crypto/hmac`, `crypto/sha256` |
| Hashes | SHA-256 | `crypto/sha256` |
| Local key protection (passphrase) | age scrypt recipient | `filippo.io/age` |
| Randomness | OS CSPRNG | `crypto/rand` |

Encodings:
- Binary values in JSON are base64url without padding (`b64`).
- Hashes are written as `sha256:<b64>`.
- **Signed JSON:** an object is signed over its exact UTF-8 bytes as stored.
  The signature is stored next to it, never inside it, so no canonical JSON
  form is needed. Readers MUST reject invalid UTF-8, duplicate object keys,
  trailing data and values outside the specified schema.

## 4. Scopes and roles

### 4.1 Scopes

| Scope | Tables | Requires read on |
|---|---|---|
| `members.core` | `members`, `roles`, `groups`, `sections`, `member_roles`, `member_groups`, `member_sections` | – |
| `members.contact` | `member_contacts` | `members.core` |
| `members.bank` | `member_bank_details` | `members.core` |
| `finance` | `finance_transactions`, `member_fees` | `members.core` |
| `inventory` | `inventory_items`, `inventory_loans` | `members.core` |
| `admin` | ACL, club settings (not a SQL table in v1) | – |

"Requires read on" exists because rows in a scope reference members. A role
that grants a scope MUST also grant read on the scopes it requires. The ACL
validator rejects ACLs that violate this.

Tables not listed are local to a device and MUST NOT be synced (see
[§22](#22-required-changes-to-the-current-data-layer) for `users` and `audit_log`).

### 4.2 Default roles

Roles are defined in the ACL ([§8](#8-access-control-list-acl)) and can be changed by an admin.
New clubs start with these:

| Role id | Label (de) | `members.core` | `members.contact` | `members.bank` | `finance` | `inventory` | `admin` |
|---|---|---|---|---|---|---|---|
| `board` | Vorstand | rw | rw | rw | rw | rw | rw |
| `treasurer` | Kassierer | r | r | rw | rw | r | – |
| `secretary` | Schriftführer | rw | rw | – | – | r | – |
| `coach` | Übungsleiter | r | – | – | – | r | – |
| `equipment_manager` | Gerätewart | r | – | – | – | rw | – |

`rw` on `admin` makes a device an **admin device**: it can sign ACL versions.

## 5. Identities and keys

### 5.1 Device keys

Each device generates on first start:
- an **Ed25519 signing key pair** (`sign_pub`, `sign_priv`)
- an **age X25519 identity** (`age_recipient`, `age_identity`)

`device_id` = first 16 bytes of `SHA-256(sign_pub)`, as lowercase base32
without padding (26 characters). Path segments use `device_id`, so it MUST
match `^[a-z2-7]{26}$`.

The private keys MUST be stored encrypted:
- in the OS keychain if available, or
- in a file encrypted with an age scrypt recipient using the user's local
  passphrase.

### 5.2 Root key and recovery kit

On club creation the first device generates the **root key**:
- an Ed25519 key pair (`root_sign_pub`, `root_sign_priv`)
- an age X25519 identity (`root_age_recipient`, `root_age_identity`)

The private parts are shown once as the **recovery kit** (text plus QR code,
designed to be printed) and then MUST be deleted from the device. The public
parts are stored in the genesis file ([§7](#7-club-genesis)).

The root key:
- signs the genesis file and ACL version 1,
- can sign any later ACL version (to recover from lost admin devices),
- receives every scope key of every epoch ([§9](#9-scope-keys-and-epochs)), so the data can always be recovered.

## 6. Storage layout

All paths are relative to the sync root of the transport.

```
club.json                                     genesis (immutable)
club.json.sig                                 root signature over club.json
acl/<version>-<hash>.json                     ACL versions (immutable)
acl/<version>-<hash>.json.sig
keys/<scope>/<epoch>/<recipient-id>.age       wrapped scope keys
requests/<device_id>.json                     join requests
requests/<device_id>.json.sig                 join-request signature
segments/<device_id>/<scope>/<seq>.seg        change segments
snapshots/<scope>/<epoch>/<hlc>-<device_id>.snap
```

- `<seq>` is a zero-padded 12-digit decimal number starting at `000000000001`.
- `<version>` is a zero-padded 8-digit decimal number.
- `<hash>` is the full lowercase hex SHA-256 of the ACL file.
- `<recipient-id>` is a `device_id` or the literal `root`.

**Every file is created once and never modified.** The only deletions are
compaction ([§15](#15-snapshots-compaction-and-erasure)) and removal of
handled join requests.

Readers MUST ignore any file whose name does not match the patterns above
exactly. This covers temporary files, `.tmp-*` files and conflict copies made
by sync clients (e.g. `0001 (conflicted copy).seg`, `0001-Konflikt.seg`).

## 7. Club genesis

`club.json`:

```json
{
  "type": "ocm.club",
  "protocol": 1,
  "club_id": "0190f5c2-7b1e-7c3a-9d2e-4b8a1f6e3c21",
  "name": "TSV Beispielstadt e.V.",
  "created_at": "2026-10-11T08:00:00Z",
  "root_sign_pub": "<b64>",
  "root_age_recipient": "age1..."
}
```

`club.json.sig` contains the Ed25519 signature of the root key over the bytes
of `club.json`, as b64.

A device joining a club MUST pin `club_id` and `root_sign_pub` locally and MUST
refuse any later `club.json` with different values.

## 8. Access control list (ACL)

### 8.1 Format

```json
{
  "type": "ocm.acl",
  "club_id": "0190f5c2-...",
  "version": 7,
  "prev": "sha256:<b64 of the bytes of version 6>",
  "author": "<device_id or 'root'>",
  "created_hlc": "<hlc>",
  "roles": {
    "board":     { "members.core": "rw", "members.contact": "rw", "members.bank": "rw", "finance": "rw", "inventory": "rw", "admin": "rw" },
    "treasurer": { "members.core": "r",  "members.contact": "r",  "members.bank": "rw", "finance": "rw", "inventory": "r" }
  },
  "devices": [
    {
      "device_id": "abcd...",
      "sign_pub": "<b64>",
      "age_recipient": "age1...",
      "person": "Erika Mustermann",
      "role": "treasurer",
      "status": "active",
      "added_in": 3
    },
    {
      "device_id": "efgh...",
      "sign_pub": "<b64>",
      "age_recipient": "age1...",
      "person": "Max Muster",
      "role": "treasurer",
      "status": "revoked",
      "added_in": 2,
      "revoked_in": 7,
      "cutoff": {
        "members.bank": { "seq": 41, "hash": "sha256:<full segment hash>" },
        "finance": { "seq": 3, "hash": "sha256:<full segment hash>" }
      }
    }
  ],
  "epochs": { "members.core": 2, "members.contact": 1, "members.bank": 2, "finance": 1, "inventory": 1, "admin": 1 },
  "clock_epochs": { "members.core": 1, "members.contact": 1, "members.bank": 1, "finance": 1, "inventory": 1, "admin": 1 },
  "key_checks": { "members.core": ["sha256:<b64 for epoch 1>", "sha256:<b64 for epoch 2>"], "...": [] },
  "min_schema": 1
}
```

`key_checks[S][e-1]` commits to `K(S,e)`: `sha256:<b64>` of the UTF-8 bytes
`"ocm.key-check.v1\n" + S + "\n" + e + "\n"` (with `e` in decimal)
followed by the 32 key bytes. A device MUST compare every key it unwraps
with this value and MUST NOT use a key that does not match. *(Added during
implementation of the identity layer, pending the review in issue #130: key
files are not signed, and a sync client replaces a file another client wrote
under the same name, so without this commitment anyone with write access to
the folder could substitute a scope key they know.)*

`.sig` file: signature by `author` (its `sign_pub`, or `root_sign_pub` for `root`).

### 8.2 Validity

ACL version `n` is valid if all of the following hold:
1. Version 1 has `prev: null` and is signed by the root key.
2. For `n > 1`, `prev` is the hash of a valid version `n-1`.
3. The author is `root`, or a device that is `active` with `rw` on `admin` in version `n-1`.
4. `epochs` never decrease from `n-1` to `n`. `key_checks[S]` has exactly
   `epochs[S]` entries, and the entries of version `n-1` are unchanged.
5. Revoked devices stay listed and are never reactivated. To re-add a person,
   add a new device.
6. Role definitions satisfy the scope dependencies of [§4.1](#41-scopes).
7. Every `device_id` equals the first 16 bytes of SHA-256 of its `sign_pub`,
   encoded as specified in [§5.1](#51-device-keys).
8. `clock_epochs` has exactly one positive integer per scope and never
   decreases. Each increase starts a new conflict-ordering generation for that
   scope ([§12](#12-hybrid-logical-clock)).
9. A revoked device's `cutoff` has exactly one `{seq, hash}` entry for every
   scope in which it had `rw` immediately before revocation. Each entry commits
   to the revoking admin's highest contiguous, verified stream position and
   complete-file hash at that time; use `{seq: 0, hash: null}` if none was
   accepted. The values are immutable after revocation.

### 8.3 Concurrent ACL edits

ACL versions are not mergeable. Admin devices MUST coordinate so only one
successor to an ACL version is published. If two valid children with the same
`prev` are discovered, clients MUST mark an ACL fork, stop advancing the ACL,
and stop accepting segments whose authorization depends on either branch.
Choosing whichever file arrived first is forbidden.

A fork can be resolved only by a new ACL version signed by the root key that
extends the branch the root explicitly selects. That root-signed extension is
final: clients MUST follow it even if they had pinned another branch, retain
the old pin for audit, reject later-arriving descendants of losing branches,
and re-evaluate/rebuild sync state under the selected chain. Until then,
syncing is fail-closed. Admin devices show a blocking warning; the root key is
in the recovery kit.

### 8.4 Rollback protection

Devices MUST remember the highest accepted ACL version and its hash. They MUST
NOT accept a lower or different chain unless it is explicitly selected by a
root-signed fork resolution under [§8.3](#83-concurrent-acl-edits).

A new device has no prior high-water mark. During enrollment, the approving
admin MUST provide an authenticated out-of-band checkpoint containing
`club_id`, `root_sign_pub`, the latest ACL version and full hash, plus
for each readable scope the SHA-256 hash of the complete newest valid
snapshot file known to the admin and its frontiers, or (if there is no
snapshot) per-stream sequence
and complete-file-hash frontiers through which the retained history can be
replayed. The checkpoint MUST identify which case applies to each scope.
Frontier sequence `0` has a `null` hash. The new device pins these values and
MUST NOT infer the current state solely from the files returned by the
transport. This protects against rollback relative to the admin's checkpoint;
folder-only transports cannot prove that no later files are being withheld.

## 9. Scope keys and epochs

- Each scope `S` and epoch `e` has a random 32-byte key `K(S,e)`.
- `K(S,e)` is wrapped with age for each **active device that may read `S`**,
  and for `root`, and stored as `keys/<S>/<e>/<recipient-id>.age`.
- When a device is granted read on `S`, it receives `K(S,e)` for **all epochs
  `e` up to the current one**, so it can read history.
- When a device loses read on `S` (revocation or role change), an admin device
  MUST:
  1. create a new epoch for `S`,
  2. wrap the new key for all remaining readers and `root`,
  3. publish the new epoch in the next ACL version.
- Writers MUST encrypt new segments with the current epoch from the newest
  ACL they know.
- A segment encrypted with an older epoch is still accepted if its author was
  authorized (see [§13.2](#132-verification)). This is intentional: devices
  that were offline during the epoch change must not lose their work.
- Epochs control confidentiality; `clock_epochs` in the ACL control conflict
  ordering. They are independent.

The `admin` scope has a key for future use (club settings). ACL files
themselves are not encrypted, because every device must be able to verify
permissions. They contain names of people with access, not member data.

## 10. Change operations

An op changes one row. Plaintext JSON:

```json
{
  "t": "members",
  "pk": { "id": "0190f5d0-..." },
  "hlc": "0000019a1b2c-0003",
  "set": { "first_name": "Erika", "last_name": "Mustermann", "number": 1042 }
}
```

| Field | Meaning |
|---|---|
| `t` | Table name. MUST be a table of the segment's scope. |
| `pk` | Primary key columns and values. Composite keys for junction tables. |
| `hlc` | Timestamp of the change ([§12](#12-hybrid-logical-clock)). |
| `set` | Columns with their new values. |
| `purge` | Optional. List of columns to erase, see [§15.3](#153-erasure-gdpr). |

Rules:
- **Insert** is an op whose `set` contains all non-null columns.
- **Update** contains only the changed columns.
- **Delete** sets `deleted_at` to an ISO-8601 UTC timestamp.
- **Restore** sets `deleted_at` to `null`.
- Ops MUST NOT contain `updated_at`. Each device derives it locally.
- Values are JSON strings, numbers, booleans or `null`. Money is integer cents.
  Dates are ISO-8601 strings.
- Rows of append-only tables are inserted once and never changed.

## 11. Segments

### 11.1 File format

```
magic       4 bytes   "OCM1"
hdr_len     4 bytes   big-endian uint32
header      hdr_len bytes, UTF-8 JSON
ciphertext  remaining bytes minus 64
signature   64 bytes  Ed25519 over (magic || hdr_len || header || ciphertext)
```

Header:

```json
{
  "club_id": "0190f5c2-...",
  "device_id": "abcd...",
  "scope": "members.core",
  "seq": 42,
  "prev": "sha256:<b64 of the complete previous segment file of this stream>",
  "epoch": 2,
  "acl_version": 7,
  "schema": 1,
  "nonce": "<b64, 24 bytes>",
  "hlc_min": "<hlc>",
  "hlc_max": "<hlc>"
}
```

- `prev` is `null` for `seq` 1.
- The hash chain makes deleted or swapped segments detectable.
- A segment's `prev` is SHA-256 of the complete preceding `.seg` file,
  including its signature. An ACL's `prev` and file-name hash are SHA-256 of
  the exact preceding ACL `.json` bytes; its `.sig` is verified separately.
  ACL hashes are full-length; a truncated digest MUST NOT be used to select or
  validate an ACL.

### 11.2 Encryption

- `plaintext` = gzip of a JSON array of ops.
- `ciphertext` = `XChaCha20-Poly1305.Seal(key = K(scope, epoch), nonce, plaintext, aad = header bytes)`.
- Using the header as associated data binds it to the content.
- A fresh 24-byte nonce MUST be obtained from the OS CSPRNG for every segment.
  It MUST NOT be derived only from `seq` or `device_id`: scope/epoch keys are
  shared across devices. A retry MUST upload the identical serialized segment,
  not encrypt the same sequence again with a new nonce.
- The exact header bytes are both AEAD associated data and part of the signed
  bytes. Readers MUST reject malformed framing. `hdr_len` MUST be at most
  65,536 bytes, ciphertext at most 16 MiB, and decompressed plaintext at most
  64 MiB; enforce each limit before allocating or decompressing untrusted input.

### 11.3 Writing

- A device collects local ops in its outbox ([§20](#20-local-state)).
- It flushes the outbox into one segment per scope:
  - after a few seconds without new changes, or
  - when the outbox exceeds 1 MiB, or
  - when the app closes.
- A segment MUST contain at least one op.
- `seq` MUST be strictly increasing by 1 per stream, without gaps. A device
  MUST persist the next `seq` before uploading. After a crash it MUST NOT reuse
  a `seq` with different content. If an upload fails, it retries with the
  identical file.

## 12. Hybrid logical clock

- An HLC is a 48-bit physical time in milliseconds plus a 16-bit counter.
- Encoding: `<12 hex digits ms>-<4 hex digits counter>`, written so that
  string order equals time order. Example: `0000019a1b2c-0003`.
- **Total order** for conflict resolution: compare
  `(clock_epoch, hlc, device_id)`, where `clock_epoch` is the value for the
  operation's scope in the ACL version named by its segment.

**On a local change:**
```
pt = now_ms()
if pt > hlc.ms: hlc = (pt, 0) else: hlc = (hlc.ms, hlc.counter + 1)
```

**On receiving an op with `r`:**
```
if r.ms <= now_ms() + MAX_DRIFT: hlc = max(hlc, r)   # then tick as above for the next local change
```

- `MAX_DRIFT` = 1 hour. Ops with `r.ms > now + MAX_DRIFT` are still applied
  (devices must converge), but the local clock does not follow them. The UI
  SHOULD show a clock warning naming the author device. This local-time check
  is informational only and MUST NOT affect acceptance or conflict resolution;
  different device clocks therefore cannot make devices accept different ops.
- There is no reliable way to prevent a malicious authorized writer from
  choosing a far-future HLC without a trusted time source. V1 accepts this
  residual risk. To recover from a future-clock attack, an admin MUST revoke
  the writer and increment `clock_epochs` for each affected scope. Newer
  generations outrank every HLC in older generations; affected values may need
  to be reviewed and re-entered. This is recovery, not prevention.
- If the counter overflows, the device waits for the next millisecond.
- **Every op gets its own tick**, also when several ops are written in the
  same database transaction. Two ops of one device therefore never share an
  HLC, and two writes to the same field within one transaction cannot tie.
- Ops in a segment MUST be ordered by ascending `hlc`. `hlc_min` and
  `hlc_max` in the header are the first and last of them.

## 13. Applying changes

### 13.1 Discovery

For each scope the device can read, the device:
1. lists `segments/*/<scope>/`,
2. for each stream, fetches segments with `seq` greater than the last applied one,
3. processes them strictly in `seq` order.

A gap (missing `seq`) stops that stream until the missing file appears.
On first sync, the device MUST load and verify the snapshot whose full file
hash is pinned for that scope in its enrollment checkpoint
([§8.4](#84-rollback-protection)), then apply segments after its recorded
positions. If the checkpoint specifies no snapshot, the device MUST replay
each stream from sequence 1 through its checkpoint frontier and verify the
pinned hash. It MUST NOT skip directly to a checkpoint frontier without the
checkpointed snapshot or a complete replay. After verifying the checkpoint,
it records those frontiers and applies later segments normally. Missing or
mismatched checkpoint data leaves the scope incomplete; the device MUST NOT
silently select an older state.

### 13.2 Verification

A segment is **accepted** only if all of these hold:
1. The file name matches the header (`device_id`, `scope`, `seq`).
2. `club_id` matches the pinned club.
3. The signature verifies with the `sign_pub` of `device_id` in the selected
   ACL chain, and `device_id` is derived from that public key. Revoked devices
   remain in the ACL, so their keys are known.
4. `prev` equals the hash of the accepted segment `seq - 1`.
5. The ACL version `acl_version` exists in the accepted chain, and in it the
   device has `rw` on `scope`, and `epoch` equals that ACL version's epoch for
   `scope`.
6. If the device is revoked in the latest unambiguous ACL, `seq` MUST NOT
   exceed `cutoff[scope].seq`. A cutoff of `0` means no segment from that
   stream is accepted. Otherwise, the complete-file hash at
   `cutoff[scope].seq` MUST equal `cutoff[scope].hash`, and the contiguous
   stream prefix MUST chain to that exact endpoint. A missing cutoff is
   invalid. Until the endpoint and its prefix can be verified, the stream
   waits and MUST NOT be reported as accepted or converged.
7. The device has `K(scope, epoch)`. If not, the stream waits.
8. Decryption succeeds, and every op's HLC is within the segment's
   `hlc_min`/`hlc_max` range; those header values equal the actual minimum and
   maximum HLCs in the batch.
9. The batch is non-empty, and every op targets a table of `scope`.
10. `schema` <= the local app's schema version. If not, the stream waits and
    the user is asked to update ([§16](#16-schema-evolution)).

Handling failures:
- **1–5, an invalid cutoff/hash in 6, 8, 9:** the segment is **rejected**.
  The stream is quarantined from that `seq` on. A file that is still being
  synced fails the signature check, so the device retries a few times before
  quarantining. Admin devices show a security warning.
- **7, 10, or missing files needed to verify a cutoff prefix:** the stream
  waits; this is not an error.

The latest unambiguous ACL determines revocation cutoffs, while the segment's
`acl_version` determines historical write permission and clock epoch. Clients
MUST defer final acceptance until they have the applicable ACL chain and
contiguous stream prefix. Combined with retroactive re-evaluation under
[§17.4](#174-revoking-a-device), this defines the same accepted set regardless
of file arrival order.

### 13.3 Apply

For each accepted segment, in one SQLite transaction:
1. Upcast ops to the local schema ([§16](#16-schema-evolution)).
2. For each op and each column in `set`, compare
   `(clock_epoch, hlc, device_id)` with the stored field clock
   ([§20](#20-local-state)). If the op is newer, write the value and the new
   field clock. Otherwise skip the column.
3. For a row that does not exist yet, insert it with the columns from `set`.
   Columns that are not set stay `NULL` or take their SQL default.
4. Set `updated_at` from the newest applied HLC of the row.
5. Foreign keys: the transaction runs with `PRAGMA defer_foreign_keys = ON`.
   If a referenced row is missing at commit (it arrives in another stream
   later), the ops that reference it go to `sync_pending`. They are retried
   after each apply round.
6. Run the uniqueness checks of [§14.2](#142-unique-values) for the touched rows.
7. Record the stream position (`seq`, file hash).

Applying the same segment twice MUST have no effect.

## 14. Conflict rules

### 14.1 Fields

- Last writer wins per field, by `(clock_epoch, hlc, device_id)`. The scope's
  clock epoch comes from the op's segment `acl_version`; a higher epoch always
  wins over a lower one, regardless of HLC.
- Deleting a row and concurrently editing another field of it do not
  conflict: the row stays deleted with the edited value. A restore with a
  newer conflict-ordering tuple brings it back.
- Append-only rows never conflict.

### 14.2 Unique values

Local SQLite MUST NOT enforce these with `UNIQUE` indexes on synced tables,
because two valid ops from different devices may collide. Instead, after
applying, the device detects duplicates among rows that are not deleted:

| Table | Column(s) |
|---|---|
| `members` | `number` |
| `member_contacts` | `email` |
| `roles`, `groups`, `sections` | `name` |

**Rule:** the row whose value was set with the smaller
`(clock_epoch, hlc, device_id)` keeps it. Every other row with the same value
is flagged in `sync_conflicts`.

The UI shows flagged rows as "Duplicate member number" (etc.):
- For member numbers it suggests the next free number.
- For names it offers to merge the two entries.

The fix is a normal change and needs `rw` on the scope.

Local edits MUST still be checked for uniqueness before saving, so the UI
prevents duplicates whenever it can see them.

### 14.3 Rows across scopes

- A `member_contacts` or `member_bank_details` row may exist while its
  member is deleted. It is hidden in the UI.
- Deleting a member in the UI MUST also write delete ops for dependent rows
  in the scopes the device can write. Rows in scopes it cannot write remain;
  a device with the right permission cleans them up (the UI lists them as
  "orphaned data").

## 15. Snapshots, compaction and erasure

### 15.1 Snapshots

- A snapshot contains the full state of one scope, including tombstones and
  field clocks (including `clock_epoch`), plus the position and complete-file
  hash of every stream it includes (`device_id → {seq, hash}`); a frontier at
  sequence `0` has a `null` hash, otherwise the hash covers the complete `.seg`
  file including its signature.
- File format: as a segment ([§11.1](#111-file-format)), with header fields
  `type: "snapshot"`, `streams` instead of `seq`/`prev`, and encryption with
  the current epoch. `acl_version` MUST name the latest unambiguous ACL known
  to the author when deriving the snapshot.
- Snapshots MUST be written by an active admin device. Other devices ignore
  snapshots from non-admin authors.
- The snapshot signature MUST be by a device that is `active`, has `rw` on
  `admin`, and has read access to the snapshot scope in both the ACL version
  named in the header and the latest unambiguous ACL. A new device MUST verify
  the genesis, the unambiguous ACL chain, the snapshot's signature,
  authorization, AEAD tag,
  schema, and stream frontiers before applying it. If frontier segments are
  still available, it MUST verify their hashes and chains. Frontiers for
  revoked devices MUST NOT exceed `cutoff[scope].seq`; a frontier at that
  sequence MUST match `cutoff[scope].hash`. The snapshot MUST contain
  only state derived from verified stream prefixes permitted by the latest ACL.
- If the complete segment prefixes are retained, a new device MUST replay them
  and compare the resulting state and field clocks with the snapshot; checking
  frontier hashes alone does not verify snapshot contents.
- A signature proves which authorized admin asserted the snapshot; it does not
  prove that the state was correctly derived. If covered segments have been
  compacted, independent verification is impossible. Trusting such a snapshot
  therefore means trusting its signing admin not to be compromised or
  malicious. A new device that cannot accept this trust assumption MUST refuse
  the snapshot and remain incomplete unless it can reconstruct the state from
  another trusted snapshot or the complete retained segment history.
- A new device MAY start from a verified snapshot and then apply segments
  strictly after the recorded positions.
- A snapshot created after a revocation MUST name an ACL version that includes
  it. An older snapshot MAY be used as a rebuild base only if its frontiers for
  revoked devices do not exceed their cutoffs and the remaining prefixes are
  replayed and checked against the cutoff hashes.
- A device that joins a club MUST load the newest valid snapshot of each scope
  it can read that is named by its authenticated enrollment checkpoint. It
  MUST verify the snapshot as specified above and then apply segments after
  the recorded positions. If the checkpoint names no snapshot for a scope,
  the device MUST replay the retained segment history through the checkpoint
  frontiers; it MUST NOT skip data that exists only in a snapshot, such as the
  initial snapshot ([§15.4](#154-initial-snapshot)).
- Admin devices SHOULD write a snapshot per scope when more than 1,000
  segments have accumulated since the last one, or once a month.

### 15.2 Compaction

An admin device MAY delete segments covered by a snapshot that is older than
the retention period (default 90 days). Devices that come back after a longer
time load the snapshot instead.

### 15.3 Erasure (GDPR)

Tombstones keep data. When personal data must be erased, e.g. a former member
after retention periods expire:
1. A device with `rw` writes an op with `purge: [columns]` (or `purge: ["*"]`
   for the whole row).
2. Every device removes these values locally and keeps only the row id and a
   tombstone. Field clocks are kept so that older ops cannot restore the
   values.
3. The next snapshot does not contain the values. Compaction removes the
   segments that still do, after the retention period.

Copies outside the protocol cannot be erased by the app: versions and the
trash in the cloud storage, backups, and devices that are offline for good.
The documentation for clubs MUST mention this.

Retention periods for finance records (§ 147 AO: 10 years) take precedence
over erasure. The UI MUST warn before purging finance data.

### 15.4 Initial snapshot

A club is usually created on a device that already holds data from local
mode. These rows were written before sync existed: they have no field clocks
from the club's history, and changes in `sync_outbox` from before the club
was created are not part of any stream.

When a club is created ([§17.1](#171-creating-a-club)), the device therefore:
1. Assigns a field clock to every column of every row in synced tables that
   has none, all with the same HLC `G` (the club creation time) and its own
   `device_id`. Existing clocks are kept.
2. Clears `sync_outbox`. Every change in it is already in the local tables
   and therefore in the snapshot.
3. Writes one snapshot per scope with epoch 1 and an empty `streams` map.
4. Sets the next `seq` of every scope to 1. The first segment after the
   snapshot has `prev: null`.

The snapshot is the starting state for all other devices. Its rows behave
like any other data: later ops change them by normal LWW, since every
later HLC is greater than `G`.

A club created on an empty database still writes the (empty) initial
snapshots, so joining devices always find one.

## 16. Schema evolution

- `schema` in segment headers is the app's sync schema version (integer).
- An app writes segments with its own schema version.
- An app MUST NOT apply segments with a higher version. It shows "Please
  update the app" and keeps working with its local data.
- Every migration that changes a synced table MUST ship an **upcaster**: a
  function that converts ops of version `n-1` to version `n` (rename columns,
  set defaults, drop removed columns).
- Columns MUST NOT be reused with a different meaning.
- `min_schema` in the ACL lets admins force old apps to update. Devices below
  it stop writing and show an update notice.

## 17. Device lifecycle

### 17.1 Creating a club

1. Generate device keys and root key.
2. Write `club.json` + signature, ACL version 1 (the device as `board`), keys
   for epoch 1 of all scopes for the device and `root`.
3. Write the initial snapshots ([§15.4](#154-initial-snapshot)).
4. Show the recovery kit. The user MUST confirm they printed or saved it.
   Then delete the root private key from the device.

**Joining with existing local data:** a device that joins a club
([§17.2](#172-inviting-a-device)) MUST start with empty synced tables.
Merging two independent data sets is not supported in version 1. If the
device holds local data, the app MUST offer to export or back it up, and
then clears the synced tables and `sync_outbox` before applying the club's
snapshots.

### 17.2 Inviting a device

Steps:
1. **Admin device creates the invite.** It generates:
   - `invite_id`, 5 random bytes,
   - `invite_secret`, 10 random bytes (80 bits).
   It stores the intended role and expiry date (default 7 days) **locally** and
   shows an **invite code**: `invite_id || invite_secret` as 24 characters of
   uppercase base32 (RFC 4648 alphabet A–Z and 2–7, no padding) in groups
   of four, e.g. `7K2P-QX3M-4TFA-W3RB-NC5H-Y6LD`.
   - The code does not contain the club id or the storage location, so it
     stays short enough to type. The joining device learns the club from
     `club.json` in the storage location the person selects.
   - For WebDAV and server storage, the app additionally offers a QR code or
     link that contains the storage URL together with the code.
   - 80 bits are enough because the secret is only used for the HMAC proof,
     invites expire after 7 days, and the safety code check follows.
2. **Access to storage:** the person must get access to the shared folder or
   WebDAV share through the storage provider. This is outside the app, and
   the app explains it.
3. **New device sends the request.** It generates its keys and writes
   `requests/<device_id>.json`:
   ```json
   { "type": "ocm.join", "club_id": "...", "invite_id": "<b64>", "device_id": "...",
     "sign_pub": "<b64>", "age_recipient": "age1...", "person": "Erika Mustermann",
     "proof": "<b64 HMAC-SHA-256(invite_secret_raw_16, join-v1 input below)>" }
   ```
   `requests/<device_id>.json.sig` contains the b64 Ed25519 signature over the
   exact request JSON bytes.
   The signature MUST cover the exact UTF-8 bytes of the complete request
   object, including the proof. `device_id` MUST be derived from `sign_pub`.
   The proof input is the byte concatenation
   `"ocm.join.v1\0" || invite_id_raw_8 || sign_pub_raw_32 ||
   age_recipient_canonical_ascii`; the HMAC key is `invite_secret_raw_16`.
   All fields MUST be decoded and validated before proof verification, and
   the proof MUST be compared in constant time.
   The request signature uses its own `sign_priv`.
4. **Admin device checks the request:**
   - The proof must verify against a known, unexpired invite.
   - The invite must be unused.
   - The admin pins the exact request bytes, their SHA-256 hash, and public keys
     being reviewed. The safety code is derived from those pinned keys. Approval
     MUST build the ACL entry from that same pinned request, without rereading
     a replaceable shared-folder path.
   - The admin displays the person's name and a six-digit safety code derived
     from `SHA-256("ocm.join-sas.v1\0" || sign_pub_raw_32 ||
     age_recipient_canonical_ascii)`: interpret the first four digest bytes as
     an unsigned big-endian integer, reduce modulo 1,000,000, and zero-pad to
     six digits. The new device displays the same code.
   - Admin and new member MUST compare the code over an independently
     authenticated channel (for example, in person or by calling a number
     already known to the admin), before approval. The shared folder is not an
     authenticated channel. A matching code binds the approval to these exact
     public keys; the displayed name alone does not.
   - HMAC proof establishes possession of the invite secret, not the identity
     of the intended person. Anyone who obtains the invite secret can make a
     valid request. The out-of-band comparison is therefore mandatory, not
     optional.
5. **Approval:** a new ACL version adds the device with the role, the
   relevant scope keys of all epochs are wrapped for it, and the request file
   is deleted. The invite is marked used. The admin provides the enrollment
   checkpoint of [§8.4](#84-rollback-protection) over the same authenticated
   out-of-band channel used for the safety-code comparison.
6. **New device starts syncing:** it verifies and pins the checkpoint, finds
   itself in the ACL, unwraps the keys and syncs.

### 17.3 Changing a role

- A new ACL version with the new role.
- If the change removes `rw` on any scope, it MUST be done by revoking the old
  device with cutoffs and inviting a new device with a new key pair. A revoked
  device is never reactivated.
- If read on a scope is removed while `rw` is not removed, start a new epoch
  for it ([§9](#9-scope-keys-and-epochs)).
- If read is added, wrap all epochs of the new scope for the device.

### 17.4 Revoking a device

1. The admin device marks the device `revoked` in a new ACL version.
2. It sets `cutoff[scope]` for every scope in which the device had `rw` to the
   highest **contiguous, verified and accepted** sequence and the complete-file
   hash at that sequence; use `{seq: 0, hash: null}` if no segment was accepted.
3. Segments the revoked device writes later are rejected by everyone, as are
   segments the admin had not yet seen.
4. A new epoch starts for every scope the device could read.

The accepted set is defined by the latest unambiguous ACL, not by the ACL a
client happened to have when a segment arrived. On learning a revocation,
clients MUST re-evaluate that device's entire stream against the final cutoff.
Any previously applied segment above the cutoff becomes invalid; clients MUST
rebuild the affected scope from a snapshot whose frontiers do not exceed the
cutoffs and the remaining accepted segment prefixes. A snapshot frontier at a
revoked device's cutoff sequence MUST match the cutoff hash. Rebuild all
derived scope state, including row values, `sync_streams`, field clocks,
history, pending ops and conflicts. An incremental apply that leaves effects
from invalidated segments is forbidden. Until the required prefixes and hash
commitments are available, the scope MUST be marked incomplete and MUST NOT
be reported as converged.

A persistent invalid file at the next sequence permanently blocks an
immutable stream; clients MUST NOT skip its sequence or overwrite it. Recovery
requires preserving the file for investigation, agreeing out of band on the
last valid sequence and hash, revoking the blocked device at that cutoff, and
enrolling a replacement device. Uncommitted local operations must be recovered
from the old device and re-entered under the replacement identity.

Lost or stolen devices are revoked the same way.

### 17.5 Recovery

If no admin device is available:
1. A device of the club (or a new device) enters the recovery kit.
2. It signs a new ACL version with the root key that makes it an admin
   device. If needed, it wraps the keys for itself (the root has all keys).
3. It deletes the root private key again.

## 18. Transports

### 18.1 Interface

```go
type Transport interface {
    // List returns the files under prefix, in any order.
    List(ctx context.Context, prefix string) ([]FileInfo, error)
    Get(ctx context.Context, path string) ([]byte, error)
    // Create writes a new file. It MUST fail with ErrExists if the path exists.
    Create(ctx context.Context, path string, data []byte) error
    // Delete is used only for compaction and join requests.
    Delete(ctx context.Context, path string) error
}
```

Because files are immutable, transports do not need locking.

### 18.2 LocalFolderTransport

- Used with a folder synced by a cloud client.
- **Create:**
  1. Write to `.tmp-<random>` in the target directory.
  2. `fsync`.
  3. Hard-link (or, if the platform does not support it, rename with an
     existence check) to the target name.
  4. Delete the temp file.
- Remote changes arrive whenever the sync client gets to them. The device
  polls with `List` (every 30 s when active) and MAY use file system
  notifications. A "Sync now" action triggers a flush and a `List`
  immediately ([§23.2](#232-other-decisions)).
- Files that are still being synced fail verification and are retried
  ([§13.2](#132-verification)).

### 18.3 WebDAVTransport

- `PROPFIND` (depth 1, recursive by directory) for `List`.
- `PUT` with `If-None-Match: *` for `Create`. `MKCOL` creates directories.
- Authentication: Basic auth with an app password, over HTTPS only. Clients
  MUST validate the server certificate and hostname, MUST reject HTTP URLs,
  and MUST NOT forward credentials to another origin or follow redirects that
  downgrade to HTTP. The password is stored in the OS keychain.

### 18.4 ServerTransport

HTTP API of the Open ClubManager server ([§19](#19-server-mode)). Same
operations, plus server-side checks.

## 19. Server mode

The server is the same Go binary started with `--serve`.

Deployments accessed over a network MUST provide HTTPS. Clients MUST validate
the server certificate; credentials and JWTs MUST NOT be sent over plaintext.
A deployment behind a reverse proxy is responsible for TLS termination.

**As a transport:**
- Devices authenticate with a challenge signed by their device key.
- `List` and `Get` return only scopes the device may read, according to the
  current ACL.
- `Create` checks the signature, the `rw` permission and `seq`/`prev`
  continuity before storing a segment.
- This adds enforcement on top of the cryptography and stops invalid data
  from reaching other devices.

**For browser access:**
- The server MAY be added to the ACL as a device (role `board` or a custom
  role). It then decrypts and maintains its own SQLite and serves the web UI.
- Browser users log in with username and password (JWT, as today). The
  server checks their rights per request. This is the only place where
  **per-group visibility** (e.g. a coach only sees their group) is possible.
- Changes made through the browser are written by the server device. The
  local audit trail records the browser user.

A server that is an ACL device can read everything its role allows. A club
that does not want this uses the server only as a transport.

## 20. Local state

Each device keeps these tables in its local SQLite. They are never synced.

| Table | Content |
|---|---|
| `sync_device` | Own `device_id`, key references, next `seq` per scope, local HLC |
| `sync_club` | Pinned `club_id`, `root_sign_pub`, enrollment checkpoint, highest accepted ACL version and hash |
| `sync_streams` | Per `(device_id, scope)`: last applied `seq`, file hash, status (`ok`, `waiting`, `quarantined`) |
| `sync_field_clock` | Per `(table, pk, column)`: `clock_epoch`, `hlc`, `device_id` |
| `sync_outbox` | Local ops not yet in a segment |
| `sync_pending` | Received ops waiting for referenced rows |
| `sync_conflicts` | Flagged duplicates ([§14.2](#142-unique-values)) |
| `sync_history` | Applied ops with author, for the change history and audit view |

Local writes go through the store package. In the same transaction as the
data change, it writes the field clocks and the ops to `sync_outbox`.

## 21. Security considerations

| Threat | Effect | Mitigation |
|---|---|---|
| Storage provider or person with folder access but no keys | Sees metadata: device ids, scopes, number and size of segments, times, names in the ACL | Content is encrypted. Names in the ACL are a known leak and could be encrypted in the `admin` scope in a later version. |
| Person with folder read access | Join requests expose the invitee's name, device id, and public keys; a malicious transport can also withhold newer files | Treat requests and ACL metadata as visible to folder readers. Do not put sensitive information in the request name; remove request files after approval. Enrollment pins an authenticated checkpoint, but folder-only sync cannot prove that later files are not being withheld. |
| Same, with write access | Can delete files, block a stream by pre-creating an invalid file at its next immutable path, or corrupt/withhold data | Signatures and hash chains detect altered files. Missing sequence numbers are detected when a later sequence or checkpoint exists; deletion of a stream's last file cannot be detected from the share alone. A blocked stream requires device replacement and recovery of unsent local operations ([§17.4](#174-revoking-a-device)). Devices keep local copies only of scopes they can read; keep independent backups. |
| Network attacker on WebDAV or remote server mode | Could steal credentials or impersonate the server over an unprotected connection | HTTPS with certificate and hostname validation is mandatory; reject downgrade redirects and do not forward credentials across origins ([§18.3](#183-webdavtransport), [§19](#19-server-mode)). |
| Member with a role | Can exfiltrate data in readable scopes and create valid changes in `rw` scopes | This is intended authority. Signatures provide attribution, not truthfulness; revocation cannot erase data already copied. |
| Revoked device | Keeps what it already had | Cannot be prevented. It cannot read new epochs, and its later writes are rejected (cutoff). |
| Authorized writer with manipulated clock | Can submit far-future HLCs that win field conflicts | No prevention is possible without trusted time. Warn, revoke the writer, and increment affected scope `clock_epochs` to permit recovery ([§12](#12-hybrid-logical-clock)); this does not undo leaked data. |
| Replay of old segments | – | `seq` and the hash chain make replays useless. |
| ACL fork or rollback | Devices could otherwise authorize different writers or rights | Fail closed on competing ACL children; only a root-signed extension resolves a fork. Pin the highest version and hash ([§8](#8-access-control-list-acl)). |
| Intercepted invite code or replaced join request | Attacker could request access as another device | HMAC alone does not authenticate the person. Require the safety-code comparison over an independently authenticated channel before approval ([§17.2](#172-inviting-a-device)). |
| Compromised admin device | Can authorize devices, sign snapshots, and write all data permitted by its scopes | Revoke it from a different admin or with the recovery kit, rotate readable scope keys, rebuild state under the new ACL, and review signed snapshots and data. Previously disclosed data cannot be recovered. |
| Admin-signed snapshot | May contain incorrect state even though its signature is valid | New devices trust the authorized signer unless they can replay retained segments; a signature is not proof of correct derivation ([§15.1](#151-snapshots)). |
| Lost recovery kit + all admin devices lost | Club data can no longer be administered | The kit is printed. The UI reminds users to keep it safe. |
| Stolen device | Local SQLite holds decrypted data | Revoke the device. The setup recommends OS disk encryption. Keys are stored in the OS keychain or encrypted with a passphrase. |

## 22. Required changes to the current data layer

These follow from this specification and should be done before or with the
sync core:

1. **New scope `inventory`** for `inventory_items` and `inventory_loans`
   (today `members.core`) in `server/internal/db/scopes.go` and
   `docs/data-model.md`.
2. **`users` becomes local.** Identity for sync is the device ([§5](#5-identities-and-keys)).
   - Password hashes MUST NOT be synced.
   - `users` stays as the login table for local and browser access.
3. **`audit_log` becomes local.** The signed segments already are the audit
   trail. The history view reads `sync_history` and local entries.
   - Today every write of every device would need `rw` on `admin` to sync
     its audit entries, which would contradict the roles.
4. **Replace `UNIQUE` indexes** on synced tables by non-unique indexes plus
   the checks of [§14.2](#142-unique-values):
   - `members_number_live`
   - `member_contacts_email_live`
   - `roles_name_live`, `groups_name_live`, `sections_name_live`
   The local pre-save check in the store package stays.
5. **Field clocks and outbox:** the store package writes them in the same
   transaction as each change ([§20](#20-local-state)).

## 23. Decisions and remaining limitations

### 23.1 Security-review decisions

1. **Clock manipulation:** no trusted time source means no sound way to reject
   future timestamps while preserving deterministic acceptance. V1 accepts
   them, warns, and provides generation-based recovery after revocation; this
   remains a risk from an authorized malicious writer.
2. **Snapshot trust:** signatures authenticate the admin who asserted a
   snapshot, not its correctness. A new device may use it only when it accepts
   that admin as trusted or can verify the snapshot by replaying the complete
   segment prefixes; without either basis it remains incomplete.
3. **Revocation:** the latest unambiguous ACL binds each cutoff to a terminal
   sequence and hash. Clients retroactively invalidate later sequence numbers
   and rebuild affected scopes instead of retaining already-applied effects.
4. **ACL forks:** there is no first-seen or hash-order winner. Clients stop
   until a root-signed extension explicitly selects a branch.
5. **New-device freshness:** an out-of-band checkpoint anchors a join to the
   approving admin's latest known state, but a folder transport cannot prove
   that later changes are not being withheld.

These rules still require implementation, interoperability tests and
cryptographic review before using this draft with real member or banking data.

### 23.2 Other decisions

6. **Names in the ACL stay readable in v1.** They are the names of club
   officials, not member data, and people with folder access belong to the
   club. Moving them into the encrypted `admin` scope is possible in a later
   protocol version.
7. **No provider sharing APIs in v1.** Giving a new member access to the
   shared folder stays outside the app. The invite screen shows an
   illustrated step-by-step guide for Nextcloud and OneDrive (others link to
   the provider's help).
8. **Latency:** the flush rules of [§11.3](#113-writing) and 30 s polling
   ([§18.2](#182-localfoldertransport)) stay. In addition, the UI MUST offer a
   "Sync now" button that flushes the outbox and lists immediately.
   Real-time sync is not a goal.
