import api from './api';
import { BACKEND_URL } from './api';

export function createEntityApi<T extends { id: number }>(endpoint: string) {
    const baseUrl = `/${endpoint}`;
    const fullUrl = `${BACKEND_URL}/${endpoint}`;

    return {
        fetchAll: async (): Promise<T[]> => {
            const res = await fetch(fullUrl);
            if (!res.ok) throw new Error(`Error fetching ${endpoint}.`);
            return res.json();
        },

        create: async (data: Omit<T, 'id'>): Promise<T> => {
            const res = await api.post<T>(baseUrl, data);
            return res.data;
        },

        update: async (data: T): Promise<T> => {
            const res = await api.put<T>(`${baseUrl}/${data.id}`, data);
            return res.data;
        },

        delete: async (data: T): Promise<void> => {
            await api.delete(`${baseUrl}/${data.id}`);
        },
    };
}
