export interface ClubEvent {
    id: string;
    ts: string;
    actor: string;
    type: string;
    entity: { kind: string; id: number };
    baseRevision: number;
    patch: Record<string, unknown>;
}

export interface LockInfo {
    owner: string;
    createdAt: string;
    expiresAt: string;
}

export interface Snapshot {
    revision: number;
    ts: string;
    entities: Record<string, Record<number, Record<string, unknown>>>;
}

export interface ClubMeta {
    name: string;
    createdAt: string;
}

export const SCHEMA_VERSION = 1;

// ── SQLite storage types ──────────────────────────────────────────────────────

export interface WriteLockInfo {
    holderId: string;
    holderLabel: string;
    acquiredAt: string;
    refreshedAt: string;
    appVersion: string;
}

export interface StorageStatus {
    dataDir: string;
    mode: 'edit' | 'readonly';
    lockHolder?: WriteLockInfo;
    lockAgeMs?: number;
}

export interface PaymentRecord {
    id: string;
    memberId: string | null;
    amountCents: number;
    currency: string;
    date: string;
    note: string | null;
    createdAt: string;
}

export interface AttachmentRecord {
    id: string;
    originalName: string;
    storedRelPath: string;
    mimeType: string;
    sizeBytes: number;
    sha256: string;
    paymentId: string | null;
    memberId: string | null;
    createdAt: string;
}
