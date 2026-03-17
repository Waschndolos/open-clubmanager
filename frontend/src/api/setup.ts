import { BACKEND_URL } from './api';

const SETUP_URL = `${BACKEND_URL}/setup`;

export async function getSetupStatus(): Promise<{ setupRequired: boolean }> {
    const res = await fetch(`${SETUP_URL}/status`);
    if (!res.ok) throw new Error("Failed to fetch setup status");
    return res.json();
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
