import { BACKEND_V2_URL } from './api';

const SETUP_URL = `${BACKEND_V2_URL}/setup`;

export type SetupDatabaseMode = 'sqlite-local' | 'mysql-shared';

export interface SetupStatusResponse {
    setupRequired: boolean;
    userCount: number;
    databaseConfigured: boolean;
    databaseMode: SetupDatabaseMode;
}

export async function getSetupStatus(): Promise<SetupStatusResponse> {
    const res = await fetch(`${SETUP_URL}/status`);
    if (!res.ok) throw new Error("Failed to fetch setup status");
    return res.json();
}

export async function configureDatabase(mode: SetupDatabaseMode, databaseUrl?: string): Promise<void> {
    const res = await fetch(`${SETUP_URL}/configure-database`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode, databaseUrl }),
    });

    if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Database configuration failed');
    }
}

export async function initializeAdmin(email: string, password: string): Promise<void> {
    const res = await fetch(`${SETUP_URL}/initialize`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
    });

    if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || "Setup failed");
    }
}
