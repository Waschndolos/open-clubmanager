import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

export interface WriteLockInfo {
    holderId: string;
    holderLabel: string;
    acquiredAt: string;
    refreshedAt: string;
    appVersion: string;
}

export interface LockStatus {
    dataDir: string;
    mode: 'edit' | 'readonly';
    lockHolder?: WriteLockInfo;
    lockAgeMs?: number;
}

const LOCK_FILE = 'write.lock.json';

/**
 * Manages a single-writer lock via an atomic file creation.
 *
 * The lock file lives directly in the data directory and follows
 * the Excel-like model: only one instance may hold the write lock;
 * all others fall back to read-only mode.
 *
 * Stale lock detection: if `refreshedAt` is older than `staleTtlMs`
 * (default 5 minutes) the lock may be broken by any instance.
 */
export class GlobalLockService {
    private readonly lockPath: string;
    private readonly holderId: string;
    private readonly holderLabel: string;
    private readonly appVersion: string;
    private readonly staleTtlMs: number;

    constructor(
        dataDir: string,
        appVersion = '1.0.0',
        staleTtlMs = 5 * 60_000
    ) {
        this.lockPath = path.join(dataDir, LOCK_FILE);
        this.holderId = `${os.hostname()}-${process.pid}`;
        this.holderLabel = os.userInfo().username;
        this.appVersion = appVersion;
        this.staleTtlMs = staleTtlMs;
    }

    /**
     * Tries to atomically create the lock file (O_EXCL / 'wx').
     * Returns true if the lock was acquired.
     * Returns false if the lock is held by someone else and is not stale.
     * Breaks a stale lock and acquires it when `refreshedAt` is older than TTL.
     */
    tryAcquire(): boolean {
        const existing = this.readLock();
        if (existing) {
            if (this.isStale(existing)) {
                this.forceDelete();
            } else {
                return false;
            }
        }

        const now = new Date().toISOString();
        const info: WriteLockInfo = {
            holderId: this.holderId,
            holderLabel: this.holderLabel,
            acquiredAt: now,
            refreshedAt: now,
            appVersion: this.appVersion,
        };

        try {
            // 'wx' flag: fail if the file already exists (atomic create).
            const fd = fs.openSync(this.lockPath, 'wx');
            fs.writeSync(fd, JSON.stringify(info, null, 2));
            fs.closeSync(fd);
            return true;
        } catch (err: unknown) {
            if ((err as NodeJS.ErrnoException).code === 'EEXIST') {
                return false;
            }
            throw err;
        }
    }

    /**
     * Refreshes the `refreshedAt` timestamp so that the lock does not
     * appear stale. Should be called on a regular interval (e.g. 30 s)
     * while the editor is open.
     */
    refresh(): void {
        const lock = this.readLock();
        if (!lock || lock.holderId !== this.holderId) return;

        const updated: WriteLockInfo = { ...lock, refreshedAt: new Date().toISOString() };
        this.writeAtomic(updated);
    }

    /**
     * Releases the lock only if it is owned by this instance.
     */
    release(): void {
        const lock = this.readLock();
        if (!lock || lock.holderId !== this.holderId) return;
        this.forceDelete();
    }

    /**
     * Returns the current lock status for UI consumption.
     */
    getStatus(dataDir: string): LockStatus {
        const lock = this.readLock();
        const isOwner = lock?.holderId === this.holderId;
        const isStale = lock ? this.isStale(lock) : false;

        if (!lock || isStale) {
            return { dataDir, mode: 'edit' };
        }

        if (isOwner) {
            return {
                dataDir,
                mode: 'edit',
                lockHolder: lock,
                lockAgeMs: Date.now() - new Date(lock.acquiredAt).getTime(),
            };
        }

        return {
            dataDir,
            mode: 'readonly',
            lockHolder: lock,
            lockAgeMs: Date.now() - new Date(lock.acquiredAt).getTime(),
        };
    }

    private isStale(lock: WriteLockInfo): boolean {
        const age = Date.now() - new Date(lock.refreshedAt).getTime();
        return age > this.staleTtlMs;
    }

    private readLock(): WriteLockInfo | null {
        if (!fs.existsSync(this.lockPath)) return null;
        try {
            const raw = fs.readFileSync(this.lockPath, 'utf-8');
            return JSON.parse(raw) as WriteLockInfo;
        } catch {
            return null;
        }
    }

    private writeAtomic(info: WriteLockInfo): void {
        const tmp = this.lockPath + '.tmp';
        fs.writeFileSync(tmp, JSON.stringify(info, null, 2), 'utf-8');
        fs.renameSync(tmp, this.lockPath);
    }

    private forceDelete(): void {
        try {
            fs.unlinkSync(this.lockPath);
        } catch {
            // Already gone – that's fine.
        }
    }
}
