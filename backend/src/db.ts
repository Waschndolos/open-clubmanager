import { PrismaClient } from '@prisma/client';
import path from "path";
import fs from "fs";


const clientCache = new Map<string, PrismaClient>();
let currentClient: PrismaClient | null = null;
let currentDbPath: string | undefined = process.env.DATABASE_URL;

export async function setActiveDb(path: string|undefined) {
    if (!path) {
        throw new Error("Database path is not defined. Please set the active database path.");
    }
    if (currentClient) {
        await currentClient.$disconnect();
    }
    currentClient = await createPrismaClientForPath(path);
    currentDbPath = path;
}

export function getCurrentDbPath(): string | undefined {
    return currentDbPath;
}

export async function getClient(): Promise<PrismaClient> {
    if (!currentClient) {
        console.log("Current Prisma client is not initialized. Initializing with default database path.");
        return await setActiveDb(process.env.DATABASE_URL).then(()=> {
            console.log("Prisma client initialized with default database path:", process.env.DATABASE_URL);
            return currentClient!;
        });

    }
    return currentClient;
}

async function createPrismaClientForPath(dbPath: string): Promise<PrismaClient> {
    if (!dbPath) {
        throw new Error("Database path is not defined. Please provide a valid database path.");
    }
    if (dbPath.startsWith("file:")) {
        // If the path starts with "file:", we assume it's already an absolute path
        dbPath = dbPath.replace("file:", "").replace(/^\/\//, ""); // Remove "file:" and leading slashes
    }

    const absolutePath = path.resolve(dbPath);

    console.log("Creating Prisma client for database path:", absolutePath);
    if (clientCache.has(absolutePath)) {
        return clientCache.get(absolutePath)!;
    }

    if (!fs.existsSync(absolutePath)) {
        fs.writeFileSync(absolutePath, '');
    }

    const dbUrl = `file:${absolutePath}`;

    console.log("dbUrl:", dbUrl);
    const client = new PrismaClient({
        datasources: {
            db: {
                url: dbUrl,
            },
        },
    });

    await client.$connect();
    clientCache.set(absolutePath, client);

    return client;
}
