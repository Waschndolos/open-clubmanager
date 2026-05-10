import 'dotenv/config';  // lädt automatisch .env
import path from "path";
import fs from "fs";
import {PrismaBetterSqlite3} from "@prisma/adapter-better-sqlite3";
import { PrismaMariaDb } from '@prisma/adapter-mariadb';
import { getAppConfig } from './v2/modules/preferences/appConfigStore.ts';
import type { PrismaClient as SqlitePrismaClient } from './generated/prisma-sqlite/client.ts';

export type DatabaseMode = 'sqlite-local' | 'mysql-shared';

type PrismaClientLike = {
    $connect(): Promise<void>;
    $disconnect(): Promise<void>;
} & SqlitePrismaClient;

type ActiveDatabase = {
    mode: DatabaseMode;
    databaseUrl: string;
};

let currentClient: PrismaClientLike | null = null;
let currentDatabase: ActiveDatabase | null = null;

function normalizeSqliteFilePath(databaseUrl: string): string {
    if (databaseUrl.startsWith('file:')) {
        return databaseUrl.replace('file:', '').replace(/^\/\//, '');
    }

    return databaseUrl;
}

function ensureSqliteFileExists(databaseUrl: string): string {
    const sqlitePath = normalizeSqliteFilePath(databaseUrl);
    const absolutePath = path.resolve(sqlitePath);

    if (!fs.existsSync(absolutePath)) {
        fs.writeFileSync(absolutePath, '');
    }

    return absolutePath;
}

async function createSqliteClient(databaseUrl: string): Promise<PrismaClientLike> {
    const absolutePath = ensureSqliteFileExists(databaseUrl);
    const adapter = new PrismaBetterSqlite3({
        url: `file:${absolutePath}`,
    });

    const module = await import('./generated/prisma-sqlite/client');
    const client = new module.PrismaClient({ adapter }) as unknown as PrismaClientLike;
    await client.$connect();
    return client;
}

async function createMysqlClient(databaseUrl: string): Promise<PrismaClientLike> {
    const module = await import('./generated/prisma-mysql/client');
    const adapter = new PrismaMariaDb(databaseUrl);
    const client = new module.PrismaClient({ adapter }) as unknown as PrismaClientLike;

    await client.$connect();
    return client;
}

export async function setActiveDatabase(mode: DatabaseMode, databaseUrl: string): Promise<void> {
    if (!databaseUrl) {
        throw new Error('Database URL is not defined.');
    }

    if (currentClient) {
        await currentClient.$disconnect();
    }

    currentClient = mode === 'sqlite-local'
        ? await createSqliteClient(databaseUrl)
        : await createMysqlClient(databaseUrl);

    currentDatabase = { mode, databaseUrl };
}

export function getCurrentDbPath(): string | undefined {
    return currentDatabase?.databaseUrl;
}

export async function setActiveDb(pathOrUrl: string | undefined): Promise<void> {
    if (!pathOrUrl) {
        throw new Error('Database path is not defined. Please set the active database path.');
    }

    const mode: DatabaseMode = /^mysqls?:\/\//i.test(pathOrUrl) ? 'mysql-shared' : 'sqlite-local';
    const databaseUrl = mode === 'sqlite-local' && !pathOrUrl.startsWith('file:')
        ? `file:${pathOrUrl}`
        : pathOrUrl;

    await setActiveDatabase(mode, databaseUrl);
}

export async function getClient(): Promise<PrismaClientLike> {
    if (!currentClient) {
        const appConfig = await getAppConfig();
        const envDatabaseUrl = process.env.DATABASE_URL;

        if (appConfig.DATABASE_URL) {
            await setActiveDatabase(appConfig.DATABASE_MODE, appConfig.DATABASE_URL);
        } else if (envDatabaseUrl) {
            await setActiveDb(envDatabaseUrl);
        } else {
            await setActiveDatabase('sqlite-local', 'file:./clubmanager.db');
        }
    }

    return currentClient!;
}

