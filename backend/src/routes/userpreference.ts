import express from 'express';
import dotenv from 'dotenv'
import { PrismaClient } from '@prisma/client';
dotenv.config()

const router = express.Router();
const prisma = new PrismaClient({
    datasources: {
        db: {
            url: process.env.DATABASE_URL
        }
    }
})

//GET /api/preferences
router.get('/', async (_, res) => {
    const sections = await prisma.userPreference.findMany();
    res.json(sections);
});

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

