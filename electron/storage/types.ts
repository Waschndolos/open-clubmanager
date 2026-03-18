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
