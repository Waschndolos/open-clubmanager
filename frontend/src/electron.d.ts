import { LockInfo } from './api/ipcTypes';

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
    finance: {
        listTransactions(): Promise<unknown[]>;
        createTransaction(data: unknown): Promise<unknown>;
        updateTransaction(data: unknown): Promise<unknown>;
        deleteTransaction(id: number): Promise<{ ok: boolean }>;
        listMemberFees(): Promise<unknown[]>;
        createMemberFee(data: unknown): Promise<unknown>;
        updateMemberFee(data: unknown): Promise<unknown>;
        deleteMemberFee(id: number): Promise<{ ok: boolean }>;
    };
    history: {
        list(): Promise<unknown[]>;
    };
}

declare global {
    interface Window {
        api?: ElectronApi;
    }
}
