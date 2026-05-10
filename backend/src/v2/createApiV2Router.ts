import { NextFunction, Request, Response, Router } from 'express';
import { createSetupRoutes } from './modules/setup/setupRoutes.ts';
import { PrismaSetupRepository } from './modules/setup/setupRepository.ts';
import { SetupService } from './modules/setup/setupService.ts';
import { readApiV2Config } from './config.ts';
import { createSystemRoutes } from './modules/system/systemRoutes.ts';
import { HttpError } from './core/httpError.ts';
import { createAuthRoutes } from './modules/auth/authRoutes.ts';
import { PrismaAuthRepository } from './modules/auth/authRepository.ts';
import { AuthService } from './modules/auth/authService.ts';
import { PrismaMembersRepository } from './modules/members/membersRepository.ts';
import { MembersService } from './modules/members/membersService.ts';
import { createMembersRoutes } from './modules/members/membersRoutes.ts';
import { createRoleRoutes } from './modules/namedEntities/roleRoutes.ts';
import { createGroupRoutes } from './modules/namedEntities/groupRoutes.ts';
import { createSectionRoutes } from './modules/namedEntities/sectionRoutes.ts';
import { createPreferencesRoutes } from './modules/preferences/preferencesRoutes.ts';
import { createValidationRoutes } from './modules/validation/validationRoutes.ts';
import { createSettingsRoutes } from './modules/settings/settingsRoutes.ts';
import { createStatisticsRoutes } from './modules/statistics/statisticsRoutes.ts';
import { createFinanceRoutes } from './modules/finance/financeRoutes.ts';
import { createHistoryRoutes } from './modules/history/historyRoutes.ts';

export function createApiV2Router(): Router {
    const router = Router();
    const config = readApiV2Config();

    const setupRepository = new PrismaSetupRepository();
    const setupService = new SetupService(setupRepository);
    const authRepository = new PrismaAuthRepository();
    const authService = new AuthService(authRepository, config);
    const membersRepository = new PrismaMembersRepository();
    const membersService = new MembersService(membersRepository);

    router.use('/system', createSystemRoutes(config));
    router.use('/setup', createSetupRoutes(setupService));
    router.use('/auth', createAuthRoutes(config, authService));
    router.use('/members', createMembersRoutes(membersService, config));
    router.use('/roles', createRoleRoutes(config));
    router.use('/groups', createGroupRoutes(config));
    router.use('/sections', createSectionRoutes(config));
    router.use('/preferences', createPreferencesRoutes());
    router.use('/validation', createValidationRoutes());
    router.use('/settings', createSettingsRoutes());
    router.use('/statistics', createStatisticsRoutes());
    router.use('/finance', createFinanceRoutes(config));
    router.use('/history', createHistoryRoutes(config));

    router.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
        if (err instanceof HttpError) {
            res.status(err.statusCode).json({ code: err.code, error: err.message });
            return;
        }

        console.error('Unhandled API v2 error:', err);
        res.status(500).json({ code: 'INTERNAL_ERROR', error: 'Internal server error' });
    });

    return router;
}

