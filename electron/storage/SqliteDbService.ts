import { createRequire } from 'module';
import type BetterSqlite3 from 'better-sqlite3';
import * as path from 'path';
import * as fs from 'fs';

export const DB_FILENAME = 'club.db';
const SCHEMA_TARGET_VERSION = 1;

// Lazy-load better-sqlite3 so that a missing or wrongly-compiled native module
// does not crash the entire Electron main process on startup.  Only the SQLite-
// specific features will be unavailable; the event-based storage keeps working.
const _require = createRequire(import.meta.url);
function loadDatabase(): typeof BetterSqlite3 {
    try {
        return _require('better-sqlite3') as typeof BetterSqlite3;
    } catch (err) {
        throw new Error(
            'better-sqlite3 native module could not be loaded. ' +
            'Run "npm run electron:rebuild" from the project root to rebuild it ' +
            'for the current Electron version, then restart the app. ' +
            `Original error: ${err}`
        );
    }
}

export interface DbMember {
    id: string;
    name: string;
    first_name: string | null;
    email: string | null;
    phone: string | null;
    birthday: string | null;
    street: string | null;
    postal_code: string | null;
    city: string | null;
    entry_date: string | null;
    exit_date: string | null;
    active: number; // SQLite booleans are stored as integers
    created_at: string;
    updated_at: string;
}

export interface DbPayment {
    id: string;
    member_id: string | null;
    amount_cents: number;
    currency: string;
    date: string;
    note: string | null;
    created_at: string;
}

export interface DbAttachment {
    id: string;
    original_name: string;
    stored_rel_path: string;
    mime_type: string;
    size_bytes: number;
    sha256: string;
    payment_id: string | null;
    member_id: string | null;
    created_at: string;
}

/**
 * Manages a SQLite database (club.db) in the chosen data directory.
 *
 * Opens in read-write mode when the caller holds the write lock, and in
 * read-only mode otherwise.  The schema is created / migrated automatically
 * on first open in read-write mode.
 */
export class SqliteDbService {
    private db: InstanceType<typeof BetterSqlite3> | null = null;
    private readonly dbPath: string;
    private readOnly = true;

    constructor(dataDir: string) {
        this.dbPath = path.join(dataDir, DB_FILENAME);
    }

    /** Opens (or re-opens) the database in the requested mode. */
    open(readOnly: boolean): void {
        if (this.db) {
            this.db.close();
            this.db = null;
        }

        this.readOnly = readOnly;

        const Database = loadDatabase();

        if (readOnly) {
            if (!fs.existsSync(this.dbPath)) {
                // No database yet – reads will return empty results.
                return;
            }
            try {
                this.db = new Database(this.dbPath, { readonly: true, fileMustExist: true });
            } catch {
                // File may have been removed between the existence check and open.
                // Leave db as null; reads will return empty results.
            }
        } else {
            this.db = new Database(this.dbPath, { readonly: false });
            this.db.pragma('journal_mode = WAL');
            this.db.pragma('foreign_keys = ON');
            this.runMigrations();
        }
    }

    close(): void {
        if (this.db) {
            this.db.close();
            this.db = null;
        }
    }

    isOpen(): boolean {
        return this.db !== null;
    }

    isReadOnly(): boolean {
        return this.readOnly;
    }

    // ── Members ───────────────────────────────────────────────────────────────

    listMembers(): DbMember[] {
        if (!this.db) return [];
        return this.db.prepare('SELECT * FROM members ORDER BY name').all() as DbMember[];
    }

    getMember(id: string): DbMember | null {
        if (!this.db) return null;
        return (this.db.prepare('SELECT * FROM members WHERE id = ?').get(id) as DbMember) ?? null;
    }

    createMember(data: Omit<DbMember, 'created_at' | 'updated_at'>): DbMember {
        this.assertWritable();
        const now = new Date().toISOString();
        this.db!.prepare(`
            INSERT INTO members
                (id, name, first_name, email, phone, birthday, street, postal_code,
                 city, entry_date, exit_date, active, created_at, updated_at)
            VALUES
                (@id, @name, @first_name, @email, @phone, @birthday, @street, @postal_code,
                 @city, @entry_date, @exit_date, @active, @created_at, @updated_at)
        `).run({ ...data, created_at: now, updated_at: now });
        return this.getMember(data.id)!;
    }

    updateMember(id: string, patch: Partial<Omit<DbMember, 'id' | 'created_at'>>): DbMember | null {
        this.assertWritable();
        const existing = this.getMember(id);
        if (!existing) return null;
        const updated = { ...existing, ...patch, updated_at: new Date().toISOString() };
        this.db!.prepare(`
            UPDATE members
            SET name = @name, first_name = @first_name, email = @email, phone = @phone,
                birthday = @birthday, street = @street, postal_code = @postal_code,
                city = @city, entry_date = @entry_date, exit_date = @exit_date,
                active = @active, updated_at = @updated_at
            WHERE id = @id
        `).run(updated);
        return this.getMember(id);
    }

    deleteMember(id: string): void {
        this.assertWritable();
        this.db!.prepare('DELETE FROM members WHERE id = ?').run(id);
    }

    // ── Payments ──────────────────────────────────────────────────────────────

    listPayments(): DbPayment[] {
        if (!this.db) return [];
        return this.db.prepare('SELECT * FROM payments ORDER BY date DESC').all() as DbPayment[];
    }

    getPayment(id: string): DbPayment | null {
        if (!this.db) return null;
        return (this.db.prepare('SELECT * FROM payments WHERE id = ?').get(id) as DbPayment) ?? null;
    }

    createPayment(data: Omit<DbPayment, 'created_at'>): DbPayment {
        this.assertWritable();
        const now = new Date().toISOString();
        this.db!.prepare(`
            INSERT INTO payments (id, member_id, amount_cents, currency, date, note, created_at)
            VALUES (@id, @member_id, @amount_cents, @currency, @date, @note, @created_at)
        `).run({ ...data, created_at: now });
        return this.getPayment(data.id)!;
    }

    updatePayment(id: string, patch: Partial<Omit<DbPayment, 'id' | 'created_at'>>): DbPayment | null {
        this.assertWritable();
        const existing = this.getPayment(id);
        if (!existing) return null;
        const updated = { ...existing, ...patch };
        this.db!.prepare(`
            UPDATE payments
            SET member_id = @member_id, amount_cents = @amount_cents, currency = @currency,
                date = @date, note = @note
            WHERE id = @id
        `).run(updated);
        return this.getPayment(id);
    }

    deletePayment(id: string): void {
        this.assertWritable();
        this.db!.prepare('DELETE FROM payments WHERE id = ?').run(id);
    }

    // ── Attachments ───────────────────────────────────────────────────────────

    listAttachments(filter?: { payment_id?: string; member_id?: string }): DbAttachment[] {
        if (!this.db) return [];
        if (filter?.payment_id) {
            return this.db
                .prepare('SELECT * FROM attachments WHERE payment_id = ? ORDER BY created_at')
                .all(filter.payment_id) as DbAttachment[];
        }
        if (filter?.member_id) {
            return this.db
                .prepare('SELECT * FROM attachments WHERE member_id = ? ORDER BY created_at')
                .all(filter.member_id) as DbAttachment[];
        }
        return this.db.prepare('SELECT * FROM attachments ORDER BY created_at').all() as DbAttachment[];
    }

    getAttachment(id: string): DbAttachment | null {
        if (!this.db) return null;
        return (this.db.prepare('SELECT * FROM attachments WHERE id = ?').get(id) as DbAttachment) ?? null;
    }

    insertAttachment(data: DbAttachment): DbAttachment {
        this.assertWritable();
        this.db!.prepare(`
            INSERT INTO attachments
                (id, original_name, stored_rel_path, mime_type, size_bytes, sha256,
                 payment_id, member_id, created_at)
            VALUES
                (@id, @original_name, @stored_rel_path, @mime_type, @size_bytes, @sha256,
                 @payment_id, @member_id, @created_at)
        `).run(data);
        return this.getAttachment(data.id)!;
    }

    deleteAttachment(id: string): void {
        this.assertWritable();
        this.db!.prepare('DELETE FROM attachments WHERE id = ?').run(id);
    }

    /**
     * Runs `fn` inside a SQLite transaction.
     * Rolls back if `fn` throws; commits otherwise.
     */
    transaction<T>(fn: () => T): T {
        this.assertWritable();
        return this.db!.transaction(fn)();
    }

    // ── Migrations ────────────────────────────────────────────────────────────

    private runMigrations(): void {
        // Ensure meta table exists so we can read/write the schema version.
        this.db!.exec(`
            CREATE TABLE IF NOT EXISTS meta (
                key   TEXT PRIMARY KEY,
                value TEXT NOT NULL
            )
        `);

        const row = this.db!.prepare('SELECT value FROM meta WHERE key = ?').get('schema_version') as
            | { value: string }
            | undefined;
        const currentVersion = row ? parseInt(row.value, 10) : 0;

        if (currentVersion < 1) {
            this.migrate_v1();
        }

        // Future migrations: if (currentVersion < 2) { this.migrate_v2(); }
    }

    private migrate_v1(): void {
        this.db!.exec(`
            CREATE TABLE IF NOT EXISTS members (
                id          TEXT PRIMARY KEY,
                name        TEXT NOT NULL,
                first_name  TEXT,
                email       TEXT,
                phone       TEXT,
                birthday    TEXT,
                street      TEXT,
                postal_code TEXT,
                city        TEXT,
                entry_date  TEXT,
                exit_date   TEXT,
                active      INTEGER NOT NULL DEFAULT 1,
                created_at  TEXT NOT NULL,
                updated_at  TEXT NOT NULL
            );

            CREATE TABLE IF NOT EXISTS payments (
                id           TEXT PRIMARY KEY,
                member_id    TEXT REFERENCES members(id) ON DELETE SET NULL,
                amount_cents INTEGER NOT NULL,
                currency     TEXT NOT NULL DEFAULT 'EUR',
                date         TEXT NOT NULL,
                note         TEXT,
                created_at   TEXT NOT NULL
            );

            CREATE TABLE IF NOT EXISTS attachments (
                id               TEXT PRIMARY KEY,
                original_name    TEXT NOT NULL,
                stored_rel_path  TEXT NOT NULL,
                mime_type        TEXT NOT NULL,
                size_bytes       INTEGER NOT NULL,
                sha256           TEXT NOT NULL,
                payment_id       TEXT REFERENCES payments(id) ON DELETE SET NULL,
                member_id        TEXT REFERENCES members(id) ON DELETE SET NULL,
                created_at       TEXT NOT NULL
            );
        `);

        this.db!.prepare("INSERT OR REPLACE INTO meta (key, value) VALUES ('schema_version', ?)").run(
            String(SCHEMA_TARGET_VERSION)
        );
    }

    private assertWritable(): void {
        if (this.readOnly || !this.db) {
            throw new Error('Database is open in read-only mode. Acquire the write lock first.');
        }
    }
}
