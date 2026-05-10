import { getClient } from '../../../db.ts';

export interface AuthUser {
    id: number;
    email: string;
    password: string;
}

export interface StoredRefreshToken {
    token: string;
    userId: number;
    expiresAt: Date;
    user: {
        id: number;
        email: string;
    };
}

export interface AuthRepository {
    findUserByEmail(email: string): Promise<AuthUser | null>;
    saveRefreshToken(token: string, userId: number, expiresAt: Date): Promise<void>;
    findRefreshToken(token: string): Promise<StoredRefreshToken | null>;
    deleteRefreshToken(token: string): Promise<void>;
    deleteAllRefreshTokensForUser(userId: number): Promise<void>;
}

export class PrismaAuthRepository implements AuthRepository {
    async findUserByEmail(email: string): Promise<AuthUser | null> {
        const prisma = await getClient();
        const user = await prisma.user.findUnique({
            where: { email },
            select: { id: true, email: true, password: true },
        });

        return user;
    }

    async saveRefreshToken(token: string, userId: number, expiresAt: Date): Promise<void> {
        const prisma = await getClient();
        await prisma.refreshToken.create({
            data: {
                token,
                userId,
                expiresAt,
            },
        });
    }

    async findRefreshToken(token: string): Promise<StoredRefreshToken | null> {
        const prisma = await getClient();

        return prisma.refreshToken.findUnique({
            where: { token },
            include: {
                user: {
                    select: {
                        id: true,
                        email: true,
                    },
                },
            },
        });
    }

    async deleteRefreshToken(token: string): Promise<void> {
        const prisma = await getClient();
        await prisma.refreshToken.deleteMany({ where: { token } });
    }

    async deleteAllRefreshTokensForUser(userId: number): Promise<void> {
        const prisma = await getClient();
        await prisma.refreshToken.deleteMany({ where: { userId } });
    }
}

