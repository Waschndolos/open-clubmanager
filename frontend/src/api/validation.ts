import { BACKEND_URL } from './api';

const BASE_URL = `${BACKEND_URL}/validation`;

export type DatabaseMode = 'sqlite-local' | 'mysql-shared';

export async function validatePath(path: string): Promise<{valid: boolean, i18nToken?: string}> {
    const res = await fetch(BASE_URL + "/check-db-path",  {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ path }),
    });
    return await res.json();
}

export async function validateDatabaseUrl(
    mode: DatabaseMode,
    databaseUrl: string
): Promise<{valid: boolean, i18nToken?: string}> {
    const res = await fetch(BASE_URL + '/check-db-url', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ mode, databaseUrl }),
    });

    return await res.json();
}
