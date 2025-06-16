import express from 'express';
import { prisma } from '../prismaClient.ts';

const router = express.Router();

router.get('/', async (_, res) => {
    const groups = await prisma.group.findMany();
    res.json(groups);
});

router.post('/', async (req, res) => {
    const { name } = req.body;
    try {
        const group = await prisma.group.create({ data: { name } });
        res.status(201).json(group);
    } catch (e) {
        res.status(400).json({ error: 'Could not create group' });
    }
});

router.put('/:id', async (req, res) => {
    const { id } = req.params;
    const { name } = req.body;
    try {
        const group = await prisma.group.update({
            where: { id: Number(id) },
            data: { name },
        });
        res.json(group);
    } catch (e) {
        res.status(400).json({ error: 'Could not update group' });
    }
});

router.delete('/:id', async (req, res) => {
    const { id } = req.params;
    try {
        await prisma.group.delete({ where: { id: Number(id) } });
        res.sendStatus(204);
    } catch (e) {
        res.status(400).json({ error: 'Could not delete group' });
    }
});

export default router;
