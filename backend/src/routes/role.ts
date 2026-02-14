import { Router, Request, Response } from 'express';
import { RoleRepository } from '../repositories/role.repository';
import { Role } from '../models/models';

const router = Router();
const repo = new RoleRepository();

/* GET /roles/:id */
router.get(
    '/:id',
    async (req: Request, res: Response): Promise<void> => {
        const role = await repo.findById(req.params.id);
        if (!role) {
            res.status(404).json({ error: 'Role not found' });
            return;
        }
        res.json(role);
    }
);

/* GET /roles – optional filtering */
router.get('/', async (req: Request, res: Response) => {
    const filter: Partial<Role> = {};
    if (req.query.name) filter.name = String(req.query.name);
    const roles = await repo.findAll(filter);
    res.json(roles);
});

/* POST /roles – upsert */
router.post('/', async (req: Request, res: Response) => {
    const payload = req.body as Role;
    const saved = await repo.upsert(payload);
    res.status(201).json(saved);
});

/* DELETE /roles/:id */
router.delete('/:id', async (req: Request, res: Response) => {
    await repo.delete(req.params.id);
    res.status(204).end();
});

export default router;