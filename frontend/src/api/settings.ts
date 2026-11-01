import api from "./api";

// Only local SQLite databases are supported.
export type DatabaseMode = 'sqlite-local';

export interface DatabaseSettings {
    mode: DatabaseMode;
    databaseUrl: string;
}

export async function saveDatabaseSettings(settings: DatabaseSettings): Promise<void> {
    await api.post('/settings/database', settings);
}

export async function getDatabaseSettings(): Promise<DatabaseSettings> {
    const res = await api.get<DatabaseSettings>('/settings/database');
    return res.data;
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