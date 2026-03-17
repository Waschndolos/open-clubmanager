import express from 'express';
import fs from 'fs/promises';
import {fileURLToPath} from 'url';
import * as path from 'path';
import { getClient } from '../db';


const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const configFilePath = path.join(__dirname, '..', '..', 'app-settings.json');

const router = express.Router();

const getConfig = async () => {
    
    const configContent = await fs.readFile(configFilePath, 'utf8');
    return JSON.parse(configContent);
}

//GET /api/preferences
router.get('/', async (_, res) => {
    const prisma = await getClient();
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
router.get('/:key', async (req, res) => {
    try {
        const { key } = req.params;
        const userId = 1;

        const prisma = await getClient();

        const preference = await prisma.userPreference.findFirst({
            where: {
                userId,
                key
            }
        });

        if (!preference) {
            res.status(404).json({ error: 'Preference not found' });
            return;
        }

        res.json({
            key: preference.key,
            value: JSON.parse(preference.value),
            updatedAt: preference.updatedAt
        });

    } catch (err) {
        console.error('Error loading preference', err);
        res.status(500).json({ error: 'Internal server error' });
    }
});


//POST /api/preferences
router.post('/', async (req, res) => {
    const { key, value } = req.body;
    const userId = 1; // später aus Auth

    try {
        const prisma = await getClient();

        const preference = await prisma.userPreference.upsert({
            where: {
                userId_key: {
                    userId,
                    key
                }
            },
            update: {
                value: JSON.stringify(value)
                // updatedAt wird automatisch gesetzt wegen @updatedAt
            },
            create: {
                userId,
                key,
                value: JSON.stringify(value)
            }
        });

        res.status(201).json(preference);
    } catch (err) {
        console.error("Error storing preference", err);
        res.status(400).json({ error: "Could not store preference" });
    }
});


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
        const prisma = await getClient();
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

