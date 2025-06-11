import express from 'express';
import { prisma } from '../prismaClient.ts';
import fs from 'fs/promises';
import { fileURLToPath } from 'url';
import * as path from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const configFilePath = path.join(__dirname, '..', '..', 'app-settings.json');

const router = express.Router();

const getConfig = async () => {
    
    const configContent = await fs.readFile(configFilePath, 'utf8');
    const config = JSON.parse(configContent);

    return config;
}

//GET /api/preferences
router.get('/', async (_, res) => {
    const sections = await prisma.userPreference.findMany();
    res.json(sections);
});

//GET /api/preferences/app
router.get('/app', async (req, res, next) => {
    try {
        const config = await getConfig();

        res.json(config);
    } catch (err) {
        console.error('Error loading preferences', err);
        res.status(500).json({ error: 'Internal server error' });
    }
});

// GET /api/preferences/app/:key
router.get('/app/:key', async (req, res) => {
    try {
        const key = req.params.key;
        const config = await getConfig();

        if (!config.hasOwnProperty(key)) {
            res.status(404).json({ error: 'Key not found in app preferences' });
            return;
        }

        res.json({ [key]: config[key] });
    } catch (err) {
        console.error('Error loading app preferences', err);
        res.status(500).json({ error: 'Internal server error' });
    }
})

//GET /api/preferences/:key
router.get('/:key', async (_req, res) => {
    try {
        const key = _req.params.key
        const userId = 1; // TODO: add once auth is there
        const preferences = await prisma.userPreference.findMany({
            where: { userId }
        });

        const parsed = preferences.map((pref) => ({
            key: key,
            value: JSON.parse(pref.value),
            updatedAt: pref.updatedAt,
        }));

        res.json(parsed[0]);
    } catch (err) {
        console.error('Error loading preferences', err);
        res.status(500).json({ error: 'Internal server error' });
    }
});

//POST /api/preferences
router.post('/', async (req, res) => {
    const { key, value } = req.body;
    try {
        const section = await prisma.userPreference.upsert({
            where: {
                id: 1 // TODO: use userID as soon as we have auth
            },
            update: {
                userId: 1, // TODO: use userID as soon as we have auth
                key: key,
                value: JSON.stringify(value),
                updatedAt: new Date()
            },
            create: {
                userId: 1, // TODO: use userID as soon as we have auth
                key: key,
                value: JSON.stringify(value),
                updatedAt: new Date()
            }
        });
        console.log("Successfully stored", section);
        res.status(201).json(section);
    } catch (e) {
        console.log("Error storing key", e);
        res.status(400).json({ error: 'Could not create section' });
    }
})

//PUT /api/preferences/app/:key
router.put('/app/:key', async (req, res) => {
    try {
        const key = req.params.key;
        const value = req.body.value;

        const config = await getConfig();

        if (!config.hasOwnProperty(key)) {
            res.status(404).json({ error: 'Key not found in app preferences' });
            return;
        }

        config[key] = value;
        await fs.writeFile(configFilePath, JSON.stringify(config, null, 2), 'utf8');

        res.json({ message: 'app preferences updated', [key]: value });
    } catch (err) {
        console.error('Error updating ', err);
        res.status(500).json({ error: 'Internal server error' });
    }
});

router.put('/:id', async (req, res) => {
    const { id } = req.params;
    const { key, value  } = req.body;
    try {
        const section = await prisma.userPreference.update({
            where: { id: Number(id) },
            data: { key: key, value: value, updatedAt: new Date() },
        });
        res.json(section);
    } catch (e) {
        res.status(400).json({ error: 'Could not update section' });
    }
});

export default router;

