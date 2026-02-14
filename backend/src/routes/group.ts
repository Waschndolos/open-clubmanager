import { Router, Request, Response } from 'express';
import { GroupRepository } from '../repositories/group.repository';
import { Group } from '../models/models';

const router = Router();
const repo = new GroupRepository();

/* GET /groups/:id */
router.get(
    '/:id',
    async (req: Request, res: Response): Promise<void> => {
        const group = await repo.findById(req.params.id);
        if (!group) {
            res.status(404).json({ error: 'Group not found' });
            return;
        }
        res.json(group);
    }
);

/* GET /groups – optional filtering */
router.get('/', async (req: Request, res: Response) => {
    const filter: Partial<Group> = {};
    if (req.query.name) filter.name = String(req.query.name);
    const groups = await repo.findAll(filter);
    res.json(groups);
});

/* POST /groups – upsert */
router.post('/', async (req: Request, res: Response) => {
    const payload = req.body as Group;
    const saved = await repo.upsert(payload);
    res.status(201).json(saved);
});

/* DELETE /groups/:id */
router.delete('/:id', async (req: Request, res: Response) => {
    await repo.delete(req.params.id);
    res.status(204).end();
});

export default router;