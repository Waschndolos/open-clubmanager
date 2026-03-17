import { BACKEND_URL } from './api';

export function createEntityApi<T extends { id: number }>(endpoint: string) {
    const baseUrl = `${BACKEND_URL}/${endpoint}`;

    return {
        fetchAll: async (): Promise<T[]> => {
            const res = await fetch(baseUrl);
            if (!res.ok) throw new Error(`Error fetching ${endpoint}.`);
            return res.json();
        },

        create: async (data: Omit<T, 'id'>): Promise<T> => {
            const res = await fetch(baseUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!res.ok) throw new Error(`Error creating ${endpoint}.`);
            return res.json();
        },

        update: async (data: T): Promise<T> => {
            const res = await fetch(`${baseUrl}/${data.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!res.ok) throw new Error(`Error updating ${endpoint}.`);
            return res.json();
        },

        delete: async (data: T): Promise<void> => {
            const res = await fetch(`${baseUrl}/${data.id}`, {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data),
            });
            if (!res.ok) throw new Error(`Error deleting ${endpoint}.`);
        },
    };
}
