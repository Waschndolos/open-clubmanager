import { getDataClient } from './clientFactory';

export type AuditLog = {
    id: number;
    action: 'CREATE' | 'UPDATE' | 'DELETE';
    entity: string;
    entityId: number;
    userId: string;
    data?: string | null;
    createdAt: string;
};

export async function fetchHistory(): Promise<AuditLog[]> {
    return getDataClient().history.list();
}
