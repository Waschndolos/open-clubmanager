import { createCrudRouter } from './crudRouter.ts';

export default createCrudRouter(prisma => prisma.clubSection, 'section');
