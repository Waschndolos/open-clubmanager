import 'dotenv/config';  // lädt automatisch .env
import path from "path";
import fs from "fs";
import {PrismaClient} from "./generated/prisma/client.ts";
import {PrismaBetterSqlite3} from "@prisma/adapter-better-sqlite3";

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
    if (dbPath.startsWith("file:")) {
        dbPath = dbPath.replace("file:", "").replace(/^\/\//, "");
    }

    const absolutePath = path.resolve(dbPath);

    if (!fs.existsSync(absolutePath)) {
        fs.writeFileSync(absolutePath, "");
    }

    const adapter = new PrismaBetterSqlite3({
        url: `file:${absolutePath}`,
    });

    const client = new PrismaClient({
        adapter,
    });

    await client.$connect();

    return client;
}

