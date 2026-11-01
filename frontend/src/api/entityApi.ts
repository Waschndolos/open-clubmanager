import api from './api';

export function createEntityApi<T extends { id: string }>(endpoint: string) {
    const baseUrl = `/${endpoint}`;

    return {
        fetchAll: async (): Promise<T[]> => {
            const res = await api.get<T[]>(baseUrl);
            return res.data;
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
