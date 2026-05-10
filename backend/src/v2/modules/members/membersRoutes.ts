import { Router } from 'express';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { MembersService } from './membersService.ts';
import { createVerifyToken } from '../auth/authMiddleware.ts';
import { ApiV2Config } from '../../config.ts';
import { parseMembersListQuery } from './membersValidation.ts';
import { AuthenticatedRequest } from '../auth/authMiddleware.ts';
import { MemberUpsertInput } from './membersTypes.ts';

export function createMembersRoutes(membersService: MembersService, config: ApiV2Config): Router {
    const router = Router();
    const verifyToken = createVerifyToken(config);

    router.get('/', asyncHandler(async (req, res) => {
        const query = parseMembersListQuery(req.query as Record<string, unknown>);
        const result = await membersService.list(query);
        res.json(result);
    }));

    router.get('/:id', asyncHandler(async (req, res) => {
        const id = Number(req.params.id);
        const result = await membersService.getById(id);
        res.setHeader('ETag', result.versionToken);
        res.json(result);
    }));

    router.post('/', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const result = await membersService.create(req.body);
        res.status(201).json(result);
    }));

    router.put('/:id', verifyToken, asyncHandler(async (req: AuthenticatedRequest, res) => {
        const id = Number(req.params.id);
        const body = req.body as Partial<MemberUpsertInput> & { expectedVersionToken?: string };
        const { expectedVersionToken, ...payload } = body;
        const result = await membersService.update(id, payload, expectedVersionToken);
        res.setHeader('ETag', result.versionToken);
        res.json(result);
    }));

    router.delete('/:id', verifyToken, asyncHandler(async (req, res) => {
        const id = Number(req.params.id);
        await membersService.delete(id);
        res.sendStatus(204);
    }));

    return router;
}

