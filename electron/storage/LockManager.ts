import * as fs from 'fs';
import * as path from 'path';
import { LockInfo } from './types.js';

const DEFAULT_TTL_MS = 10 * 60 * 1000; // 10 minutes

export class LockManager {
    private locksDir: string;

    constructor(clubFolder: string) {
        this.locksDir = path.join(clubFolder, 'locks');
    }

    private lockPath(resource: string): string {
        const safe = resource.replace(/[^a-zA-Z0-9_-]/g, '_');
        return path.join(this.locksDir, `${safe}.lock`);
    }

    private readLock(resource: string): LockInfo | null {
        const lockFile = this.lockPath(resource);
        if (!fs.existsSync(lockFile)) return null;
        try {
            const raw = fs.readFileSync(lockFile, 'utf-8');
            return JSON.parse(raw) as LockInfo;
        } catch {
            return null;
        }
    }

    private writeLock(resource: string, info: LockInfo): void {
        const tmpPath = this.lockPath(resource) + '.tmp';
        fs.writeFileSync(tmpPath, JSON.stringify(info, null, 2), 'utf-8');
        fs.renameSync(tmpPath, this.lockPath(resource));
    }

    isLocked(resource: string): boolean {
        const lock = this.readLock(resource);
        if (!lock) return false;
        if (new Date(lock.expiresAt) < new Date()) {
            return false; // stale lock
        }
        return true;
    }

    getLock(resource: string): LockInfo | null {
        const lock = this.readLock(resource);
        if (!lock) return null;
        if (new Date(lock.expiresAt) < new Date()) {
            return null; // stale lock
        }
        return lock;
    }

    acquire(resource: string, owner: string, ttlMs: number = DEFAULT_TTL_MS): boolean {
        const existing = this.getLock(resource);
        if (existing && existing.owner !== owner) {
            return false; // locked by someone else
        }
        const now = new Date();
        const info: LockInfo = {
            owner,
            createdAt: now.toISOString(),
            expiresAt: new Date(now.getTime() + ttlMs).toISOString(),
        };
        this.writeLock(resource, info);
        return true;
    }

    renew(resource: string, owner: string, ttlMs: number = DEFAULT_TTL_MS): boolean {
        const existing = this.getLock(resource);
        if (!existing || existing.owner !== owner) {
            return false;
        }
        const now = new Date();
        const info: LockInfo = {
            owner,
            createdAt: existing.createdAt,
            expiresAt: new Date(now.getTime() + ttlMs).toISOString(),
        };
        this.writeLock(resource, info);
        return true;
    }

    release(resource: string, owner: string): boolean {
        const existing = this.readLock(resource);
        if (!existing || existing.owner !== owner) {
            return false;
        }
        try {
            fs.unlinkSync(this.lockPath(resource));
        } catch {
            // already gone
        }
        return true;
    }

    forceRelease(resource: string): void {
        try {
            fs.unlinkSync(this.lockPath(resource));
        } catch {
            // already gone
        }
    }
}
