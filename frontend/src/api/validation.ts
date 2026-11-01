import api from './api';

export async function validatePath(path: string): Promise<{valid: boolean, i18nToken?: string}> {
    const res = await api.post<{valid: boolean, i18nToken?: string}>('/validation/check-db-path', { path });
    return res.data;
}

