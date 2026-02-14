// src/routes/userPreference.ts
import {Request, Response, Router} from 'express';
import {UserPreferenceRepository} from '../repositories/userPreference.repository';
import {UserPreference} from '../models/models';

const router = Router();
const repo = new UserPreferenceRepository();

/* GET /preferences/:id */
router.get(
    '/:id',
    async (req: Request, res: Response): Promise<void> => {
        const pref = await repo.findById(req.params.id);
        if (!pref) {
            res.status(404).json({ error: 'Preference not found' });
            return;
        }
        res.json(pref);
    }
);

/* GET /preferences – optional filter (userId, key) */
router.get('/', async (req: Request, res: Response) => {
    const filter: Partial<UserPreference> = {};
    if (req.query.userId) filter.userId = Number(req.query.userId);
    if (req.query.key) filter.key = String(req.query.key);
    const prefs = await repo.findAll(filter);
    res.json(prefs);
});

/* POST /preferences – upsert (unique on userId+key) */
router.post('/', async (req: Request, res: Response) => {
    const payload = req.body as UserPreference;
    const saved = await repo.upsert(payload);
    res.status(201).json(saved);
});

/* DELETE /preferences/:id */
router.delete('/:id', async (req: Request, res: Response) => {
    await repo.delete(req.params.id);
    res.status(204).end();
});

export default router;