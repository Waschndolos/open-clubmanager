import { PrismaClient } from '@prisma/client';
import fs from 'fs/promises';
import { fileURLToPath } from 'url';
import dotenv from 'dotenv';
import * as path from 'path';

dotenv.config();

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

async function initializePrisma() {
    let databaseUrl = process.env.DATABASE_URL;
    const configFilePath = path.join(__dirname, '..', 'app-settings.json');

    try {
        // Check if config file exists
        try {
            await fs.access(configFilePath);
        } catch {
            console.log('app-settings.json not found, creating one');
            const settings = {
                DATABASE_URL: process.env.DATABASE_URL || ''
            };
            await fs.writeFile(configFilePath, JSON.stringify(settings, null, 2), 'utf8');
            console.log('app-settings.json created');
        }

        const configContent = await fs.readFile(configFilePath, 'utf8');
        const config = JSON.parse(configContent);
        
        // Ensure proper object structure
        const safeConfig = Object.assign(Object.create(Object.prototype), config);
        
        if (safeConfig.DATABASE_URL && safeConfig.DATABASE_URL.trim() !== '') {
            databaseUrl = safeConfig.DATABASE_URL;
            console.log('Using DATABASE_URL from app-settings.json');
        }
    } catch (error) {
        console.error('Error loading or creating app-settings.json:', error);
        databaseUrl = process.env.DATABASE_URL;
    }

    if (!databaseUrl || databaseUrl.trim() === '') {
        throw new Error('DATABASE_URL is required');
    }

    return new PrismaClient({
        datasources: {
            db: {
                url: databaseUrl
            }
        }
    });
}

// Export a promise that resolves to the Prisma client
export const prismaPromise = initializePrisma();

// For immediate use (but handle the promise)
export const prisma = await initializePrisma();