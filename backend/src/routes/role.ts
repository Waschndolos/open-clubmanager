import express from 'express';
import { prisma } from '../prismaClient.ts';

const router = express.Router();

router.get('/', async (_, res) => {
    const roles = await prisma.role.findMany();
    res.json(roles);
});

router.post('/', async (req, res) => {
    const { name } = req.body;
    try {
        const role = await prisma.role.create({ data: { name } });
        res.status(201).json(role);
    } catch (e) {
        res.status(400).json({ error: 'Could not create role' });
    }
});

router.put('/:id', async (req, res) => {
    const { id } = req.params;
    const { name } = req.body;
    try {
        const role = await prisma.role.update({
            where: { id: Number(id) },
            data: { name },
        });
        res.json(role);
    } catch (e) {
        res.status(400).json({ error: 'Could not update role' });
    }
});

router.delete('/:id', async (req, res) => {
    const { id } = req.params;
    try {
        await prisma.role.delete({ where: { id: Number(id) } });
        res.sendStatus(204);
    } catch (e) {
        res.status(400).json({ error: 'Could not delete role' });
    }
});

export default router;
