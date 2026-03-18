import api from './api';

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
    const res = await api.get<AuditLog[]>('/history');
    return res.data;
}
