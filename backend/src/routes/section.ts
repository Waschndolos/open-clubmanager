import express from 'express';
import { prisma } from '../prismaClient.ts';

const router = express.Router();

router.get('/', async (_, res) => {
    const sections = await prisma.clubSection.findMany();
    res.json(sections);
});

router.post('/', async (req, res) => {
    const { name } = req.body;
    try {
        const section = await prisma.clubSection.create({ data: { name } });
        res.status(201).json(section);
    } catch (e) {
        res.status(400).json({ error: 'Could not create section' });
    }
});

router.put('/:id', async (req, res) => {
    const { id } = req.params;
    const { name } = req.body;
    try {
        const section = await prisma.clubSection.update({
            where: { id: Number(id) },
            data: { name },
        });
        res.json(section);
    } catch (e) {
        res.status(400).json({ error: 'Could not update section' });
    }
});

router.delete('/:id', async (req, res) => {
    const { id } = req.params;
    try {
        await prisma.clubSection.delete({ where: { id: Number(id) } });
        res.sendStatus(204);
    } catch (e) {
        res.status(400).json({ error: 'Could not delete section' });
    }
});

export default router;
