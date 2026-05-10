import { getClient } from '../../../db.ts';

export interface SetupRepository {
    countUsers(): Promise<number>;
    createInitialAdmin(email: string, passwordHash: string): Promise<void>;
}

export class PrismaSetupRepository implements SetupRepository {
    async countUsers(): Promise<number> {
        const prisma = await getClient();
        return prisma.user.count();
    }

    async createInitialAdmin(email: string, passwordHash: string): Promise<void> {
        const prisma = await getClient();
        await prisma.user.create({
            data: {
                email,
                password: passwordHash,
            },
        });
    }
}

