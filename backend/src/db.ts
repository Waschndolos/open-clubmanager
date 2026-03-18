import 'dotenv/config';  // lädt automatisch .env
import pathModule from "path";
import fs from "fs";
import os from "os";
import { execSync } from "child_process";
import { fileURLToPath } from "url";
import {PrismaClient} from "./generated/prisma/client.ts";
import {PrismaBetterSqlite3} from "@prisma/adapter-better-sqlite3";

const __filename = fileURLToPath(import.meta.url);
const __dirname = pathModule.dirname(__filename);

const CONFIG_DIR = pathModule.join(os.homedir(), ".clubmanager");
const CONFIG_FILE = pathModule.join(CONFIG_DIR, "config.json");

let currentClient: PrismaClient | null = null;
let currentDbPath: string | undefined = undefined;

function saveDbPathToConfig(dbPath: string): void {
    if (!fs.existsSync(CONFIG_DIR)) {
        fs.mkdirSync(CONFIG_DIR, { recursive: true });
    }
    fs.writeFileSync(CONFIG_FILE, JSON.stringify({ dbPath }), "utf-8");
}

function loadDbPathFromConfig(): string | undefined {
    if (fs.existsSync(CONFIG_FILE)) {
        try {
            const config = JSON.parse(fs.readFileSync(CONFIG_FILE, "utf-8"));
            return config.dbPath as string | undefined;
        } catch {
            return undefined;
        }
    }
    return undefined;
}

function runMigrations(absoluteDbPath: string): void {
    const env = { ...process.env, DATABASE_URL: `file:${absoluteDbPath}` };
    const prismaBin = pathModule.join(__dirname, "..", "..", "node_modules", ".bin", "prisma");
    const schemaPath = pathModule.join(__dirname, "..", "..", "prisma", "schema.prisma");
    const configPath = pathModule.join(__dirname, "..", "..", "prisma.config.ts");

    const binPath = fs.existsSync(prismaBin) ? prismaBin : "prisma";
    const configFlag = fs.existsSync(configPath)
        ? `--config="${configPath}"`
        : `--schema="${schemaPath}"`;

    execSync(`"${binPath}" migrate deploy ${configFlag}`, {
        env,
        cwd: pathModule.join(__dirname, "..", ".."),
        stdio: "inherit",
    });
}

export function isDbConfigured(): boolean {
    return currentClient !== null || !!process.env.DATABASE_URL;
}

export async function setActiveDb(dbPath: string | undefined): Promise<void> {
    if (!dbPath) {
        throw new Error("Database path is not defined. Please set the active database path.");
    }
    if (currentClient) {
        await currentClient.$disconnect();
        currentClient = null;
    }

    const resolvedPath = resolveAbsolutePath(dbPath);

    if (!fs.existsSync(resolvedPath)) {
        fs.writeFileSync(resolvedPath, "");
    }

    runMigrations(resolvedPath);

    currentClient = await createPrismaClientForPath(resolvedPath);
    currentDbPath = dbPath;

    saveDbPathToConfig(resolvedPath);
}

export function getCurrentDbPath(): string | undefined {
    return currentDbPath;
}

export async function getClient(): Promise<PrismaClient> {
    if (!currentClient) {
        const dbUrl = process.env.DATABASE_URL || loadDbPathFromConfig();
        if (dbUrl) {
            console.log("Prisma client not initialized. Initializing with database path:", dbUrl);
            await setActiveDb(dbUrl);
            return currentClient!;
        }
        throw new Error("Database is not configured. Please set up the database path first.");
    }
    return currentClient;
}

function resolveAbsolutePath(dbPath: string): string {
    let resolved = dbPath;
    if (resolved.startsWith("file:")) {
        resolved = resolved.replace("file:", "").replace(/^\/\//, "");
    }
    return pathModule.resolve(resolved);
}

async function createPrismaClientForPath(absoluteDbPath: string): Promise<PrismaClient> {
    const adapter = new PrismaBetterSqlite3({
        url: `file:${absoluteDbPath}`,
    });

    const client = new PrismaClient({
        adapter,
    });

    await client.$connect();

    return client;
}

// Auto-initialize from persisted config or environment variable on module load
(async () => {
    const storedPath = loadDbPathFromConfig();
    const envPath = process.env.DATABASE_URL;
    const initPath = storedPath || envPath;
    if (initPath && !currentClient) {
        try {
            await setActiveDb(initPath);
            console.log("Database initialized from stored config:", initPath);
        } catch (err) {
            console.warn("Could not auto-initialize database:", err);
        }
    }
})();

