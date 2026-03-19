import { LockInfo } from './api/ipcTypes';

export interface WriteLockInfo {
    holderId: string;
    holderLabel: string;
    acquiredAt: string;
    refreshedAt: string;
    appVersion: string;
}

export interface StorageStatus {
    dataDir: string | null;
    mode: 'edit' | 'readonly';
    lockHolder?: WriteLockInfo;
    lockAgeMs?: number;
}

export interface ElectronApi {
    club: {
        selectFolder(): Promise<string | null>;
        getFolder(): Promise<string | null>;
        initFolder(): Promise<{ ok: boolean }>;
    };
    members: {
        list(): Promise<unknown[]>;
        get(id: number): Promise<unknown>;
        create(data: unknown): Promise<unknown>;
        update(id: number, patch: unknown): Promise<unknown>;
        delete(ids: number[]): Promise<{ ok: boolean }>;
        lock(id: number, owner: string): Promise<{ acquired: boolean; lock: LockInfo | null }>;
        unlock(id: number, owner: string): Promise<{ released: boolean }>;
        getLock(id: number): Promise<LockInfo | null>;
    };
    roles: {
        list(): Promise<unknown[]>;
        create(data: unknown): Promise<unknown>;
        update(data: unknown): Promise<unknown>;
        delete(id: number): Promise<{ ok: boolean }>;
    };
    groups: {
        list(): Promise<unknown[]>;
        create(data: unknown): Promise<unknown>;
        update(data: unknown): Promise<unknown>;
        delete(id: number): Promise<{ ok: boolean }>;
    };
    sections: {
        list(): Promise<unknown[]>;
        create(data: unknown): Promise<unknown>;
        update(data: unknown): Promise<unknown>;
        delete(id: number): Promise<{ ok: boolean }>;
    };
    storage: {
        getStatus(): Promise<StorageStatus>;
        requestEditMode(): Promise<{ acquired: boolean; status: StorageStatus }>;
        releaseEditMode(): Promise<{ ok: boolean }>;
        exportBackup(): Promise<{ zipPath: string }>;
    };
    payments: {
        list(): Promise<unknown[]>;
        create(data: unknown): Promise<unknown>;
        update(data: unknown): Promise<unknown>;
        delete(id: string): Promise<{ ok: boolean }>;
    };
    attachments: {
        add(data: unknown): Promise<unknown>;
        list(filter?: unknown): Promise<unknown[]>;
        open(id: string): Promise<{ ok: boolean }>;
    };
}

declare global {
    interface Window {
        api?: ElectronApi;
    }
}
