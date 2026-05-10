import { ApiV2Config } from '../../config.ts';
import { createNamedEntityRoutes } from './namedEntityRoutes.ts';

export function createRoleRoutes(config: ApiV2Config) {
    return createNamedEntityRoutes(config, (prisma) => prisma.role, 'role');
}

