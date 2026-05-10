import {BACKEND_URL} from "./api";

const BASE_URL = `${BACKEND_URL}/settings`;

export type DatabaseMode = 'sqlite-local' | 'mysql-shared';

export interface DatabaseSettings {
    mode: DatabaseMode;
    databaseUrl: string;
}

export async function saveDatabaseSettings(settings: DatabaseSettings): Promise<void> {
    const res = await fetch(`${BASE_URL}/database`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
    });

    if (!res.ok) throw new Error(await res.text());
}

export async function getDatabaseSettings(): Promise<DatabaseSettings> {
    const res = await fetch(`${BASE_URL}/database`);
    if (!res.ok) throw new Error(await res.text());
    return res.json();
}

export async function saveDbPath(dbPath: string) {
    return saveDatabaseSettings({
        mode: 'sqlite-local',
        databaseUrl: dbPath.startsWith('file:') ? dbPath : `file:${dbPath}`,
    });
}

export async function getDbPath(): Promise<string> {
    const data = await getDatabaseSettings();
    return stripFilePrefix(data.databaseUrl);
}

function stripFilePrefix(path: string): string {
    if (path.startsWith("file:")) {
        return path.replace("file:", "").replace(/^\/\//, ""); // Remove "file:" and leading slashes
    }
    return path;
}