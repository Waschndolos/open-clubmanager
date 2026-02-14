// src/routes/clubSection
import { Router, Request, Response } from 'express';
import { ClubSectionRepository } from '../repositories/clubSection.repository';
import { ClubSection } from '../models/models';

const router = Router();
const repo = new ClubSectionRepository();

/* -------------------------------------------------
   GET /sections/:id  – fetch a single club section
   ------------------------------------------------- */
router.get(
    '/:id',
    async (req: Request, res: Response): Promise<void> => {
        const section = await repo.findById(req.params.id);
        if (!section) {
            res.status(404).json({ error: 'Section not found' });
            return;
        }
        res.json(section);
    }
);

/* -------------------------------------------------
   GET /sections  – list all sections (optional filter)
   ------------------------------------------------- */
router.get('/', async (req: Request, res: Response) => {
    const filter: Partial<ClubSection> = {};
    if (req.query.name) filter.name = String(req.query.name);
    const sections = await repo.findAll(filter);
    res.json(sections);
});

/* -------------------------------------------------
   POST /sections  – create or update (upsert) a section
   ------------------------------------------------- */
router.post('/', async (req: Request, res: Response) => {
    const payload = req.body as ClubSection;
    const saved = await repo.upsert(payload);
    res.status(201).json(saved);
});

/* -------------------------------------------------
   DELETE /sections/:id  – remove a section
   ------------------------------------------------- */
router.delete('/:id', async (req: Request, res: Response) => {
    await repo.delete(req.params.id);
    res.status(204).end();
});

export default router;