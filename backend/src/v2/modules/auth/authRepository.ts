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

export interface StoredPasswordResetToken {
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
    findUserById(id: number): Promise<AuthUser | null>;
    updateUserPassword(userId: number, passwordHash: string): Promise<void>;
    saveRefreshToken(token: string, userId: number, expiresAt: Date): Promise<void>;
    findRefreshToken(token: string): Promise<StoredRefreshToken | null>;
    deleteRefreshToken(token: string): Promise<void>;
    deleteAllRefreshTokensForUser(userId: number): Promise<void>;
    savePasswordResetToken(token: string, userId: number, expiresAt: Date): Promise<void>;
    findPasswordResetToken(token: string): Promise<StoredPasswordResetToken | null>;
    deletePasswordResetToken(token: string): Promise<void>;
    deleteAllPasswordResetTokensForUser(userId: number): Promise<void>;
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

    async findUserById(id: number): Promise<AuthUser | null> {
        const prisma = await getClient();
        const user = await prisma.user.findUnique({
            where: { id },
            select: { id: true, email: true, password: true },
        });

        return user;
    }

    async updateUserPassword(userId: number, passwordHash: string): Promise<void> {
        const prisma = await getClient();
        await prisma.user.update({
            where: { id: userId },
            data: { password: passwordHash },
        });
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

    async savePasswordResetToken(token: string, userId: number, expiresAt: Date): Promise<void> {
        const prisma = await getClient();
        await prisma.passwordResetToken.create({
            data: {
                token,
                userId,
                expiresAt,
            },
        });
    }

    async findPasswordResetToken(token: string): Promise<StoredPasswordResetToken | null> {
        const prisma = await getClient();

        return prisma.passwordResetToken.findUnique({
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

    async deletePasswordResetToken(token: string): Promise<void> {
        const prisma = await getClient();
        await prisma.passwordResetToken.deleteMany({ where: { token } });
    }

    async deleteAllPasswordResetTokensForUser(userId: number): Promise<void> {
        const prisma = await getClient();
        await prisma.passwordResetToken.deleteMany({ where: { userId } });
    }
}

