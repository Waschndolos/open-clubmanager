import { Router } from 'express';
import { getClient } from '../../../db.ts';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { getAppConfig, setAppConfig } from './appConfigStore.ts';

function normalizeParamKey(rawKey: string | string[] | undefined): string {
    if (Array.isArray(rawKey)) {
        return rawKey[0] ?? '';
    }
    return rawKey ?? '';
}

export function createPreferencesRoutes(): Router {
    const router = Router();

    router.get('/', asyncHandler(async (_req, res) => {
        const prisma = await getClient();
        const preferences = await prisma.userPreference.findMany();
        res.json(preferences);
    }));

    router.get('/app', asyncHandler(async (_req, res) => {
        const config = await getAppConfig();
        res.json(config);
    }));

    router.get('/app/:key', asyncHandler(async (req, res) => {
        const key = normalizeParamKey(req.params.key);
        const config = await getAppConfig();
        const configRecord = config as unknown as Record<string, unknown>;

        if (!Object.prototype.hasOwnProperty.call(configRecord, key)) {
            res.status(404).json({ error: 'Key not found in app preferences' });
            return;
        }

        res.json({ [key]: configRecord[key] });
    }));

    router.get('/:key', asyncHandler(async (req, res) => {
        const key = normalizeParamKey(req.params.key);
        const userId = 1;
        const prisma = await getClient();

        const preference = await prisma.userPreference.findFirst({
            where: { userId, key },
        });

        if (!preference) {
            res.status(404).json({ error: 'Preference not found' });
            return;
        }

        res.json({
            key: preference.key,
            value: JSON.parse(preference.value),
            updatedAt: preference.updatedAt,
        });
    }));

    router.post('/', asyncHandler(async (req, res) => {
        const { key, value } = req.body as { key?: string; value?: unknown };
        const userId = 1;
        const prisma = await getClient();

        const preference = await prisma.userPreference.upsert({
            where: {
                userId_key: {
                    userId,
                    key: key ?? '',
                },
            },
            update: {
                value: JSON.stringify(value),
            },
            create: {
                userId,
                key: key ?? '',
                value: JSON.stringify(value),
            },
        });

        res.status(201).json(preference);
    }));

    router.put('/app/:key', asyncHandler(async (req, res) => {
        const key = normalizeParamKey(req.params.key);
        const value = (req.body as { value?: unknown }).value;
        const config = await getAppConfig();
        const configRecord = config as unknown as Record<string, unknown>;

        if (!Object.prototype.hasOwnProperty.call(configRecord, key)) {
            res.status(404).json({ error: 'Key not found in app preferences' });
            return;
        }

        configRecord[key] = value;
        await setAppConfig({
            DATABASE_URL: String(configRecord.DATABASE_URL ?? ''),
            DATABASE_MODE: configRecord.DATABASE_MODE === 'mysql-shared' ? 'mysql-shared' : 'sqlite-local',
        });
        res.json({ message: 'app preferences updated', [key]: value });
    }));

    router.put('/:id', asyncHandler(async (req, res) => {
        const id = Number(req.params.id);
        const { key, value } = req.body as { key?: string; value?: string };
        const prisma = await getClient();

        const preference = await prisma.userPreference.update({
            where: { id },
            data: { key: key ?? '', value: value ?? '', updatedAt: new Date() },
        });

        res.json(preference);
    }));

    return router;
}

