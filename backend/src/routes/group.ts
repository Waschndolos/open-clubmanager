import { createCrudRouter } from './crudRouter.ts';

export default createCrudRouter(prisma => prisma.group, 'group');
