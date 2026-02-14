import {Request, Response, Router} from 'express';
import {MemberRepository} from '../repositories/member.repository';
import {Member} from '../models/models';

const router = Router();
const repo = new MemberRepository();

/* GET /members/:id – retrieve a single member */
router.get(
    '/:id',
    async (req: Request, res: Response): Promise<void> => {
        const member = await repo.findById(req.params.id);
        if (!member) {
            res.status(404).json({error: 'Member not found'});
            return; // stop execution – no value returned
        }
        res.json(member);
    }
);

/* GET /members – list members, optional filtering via query parameters */
router.get('/', async (req: Request, res: Response) => {
    const filter: Partial<Member> = {};
    if (req.query.email) filter.email = String(req.query.email);
    if (req.query.firstName) filter.firstName = String(req.query.firstName);
    // Add more filters as needed
    const members = await repo.findAll(filter);
    res.json(members);
});

/* POST /members – create or update (upsert) a member */
router.post('/', async (req: Request, res: Response) => {
    const payload = req.body as Member;
    const saved = await repo.upsert(payload);
    res.status(201).json(saved);
});

/* DELETE /members/:id */
router.delete('/:id', async (req: Request, res: Response) => {
    await repo.delete(req.params.id);
    res.status(204).end();
});

export default router;