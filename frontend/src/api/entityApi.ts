import api from './api';
import { BACKEND_URL } from './api';
import { isElectronFolderMode } from '../lib/environment';
import { getDataClient } from './clientFactory';

type EntityClientLike<T extends { id: number }> = {
    list?: () => Promise<T[]>;
    create?: (data: Omit<T, 'id'>) => Promise<T>;
    update?: (data: T) => Promise<T>;
    delete?: (data: T) => Promise<void>;
};

export function createEntityApi<T extends { id: number }>(endpoint: string) {
    const baseUrl = `/${endpoint}`;
    const fullUrl = `${BACKEND_URL}/${endpoint}`;

    function ipcEntityClient(): EntityClientLike<T> | undefined {
        const client = getDataClient() as unknown as Record<string, unknown>;
        return client[endpoint] as EntityClientLike<T> | undefined;
    }

    return {
        fetchAll: async (): Promise<T[]> => {
            if (isElectronFolderMode()) {
                const entityClient = ipcEntityClient();
                if (entityClient?.list) {
                    return entityClient.list();
                }
            }
            const res = await fetch(fullUrl);
            if (!res.ok) throw new Error(`Error fetching ${endpoint}.`);
            return res.json();
        },

        create: async (data: Omit<T, 'id'>): Promise<T> => {
            if (isElectronFolderMode()) {
                const entityClient = ipcEntityClient();
                if (entityClient?.create) {
                    return entityClient.create(data);
                }
            }
            const res = await api.post<T>(baseUrl, data);
            return res.data;
        },

        update: async (data: T): Promise<T> => {
            if (isElectronFolderMode()) {
                const entityClient = ipcEntityClient();
                if (entityClient?.update) {
                    return entityClient.update(data);
                }
            }
            const res = await api.put<T>(`${baseUrl}/${data.id}`, data);
            return res.data;
        },

        delete: async (data: T): Promise<void> => {
            if (isElectronFolderMode()) {
                const entityClient = ipcEntityClient();
                if (entityClient?.delete) {
                    return entityClient.delete(data);
                }
            }
            await api.delete(`${baseUrl}/${data.id}`);
        },
    };
}
