import bcrypt from 'bcrypt';
import jwt from 'jsonwebtoken';
import { ApiV2Config } from '../../config.ts';
import { HttpError } from '../../core/httpError.ts';
import { AuthRepository } from './authRepository.ts';

export interface LoginResult {
    accessToken: string;
    refreshToken: string;
}

export interface RefreshResult {
    accessToken: string;
}

const ACCESS_TOKEN_TTL_SECONDS = 15 * 60;
const REFRESH_TOKEN_TTL_DAYS = 7;

function createExpiresAtFromNow(days: number): Date {
    const result = new Date();
    result.setDate(result.getDate() + days);
    return result;
}

export class AuthService {
    constructor(
        private readonly authRepository: AuthRepository,
        private readonly config: ApiV2Config
    ) {}

    async login(email: string, password: string): Promise<LoginResult> {
        if (!email || !password) {
            throw new HttpError(400, 'AUTH_INVALID_INPUT', 'Email and password are required.');
        }

        const normalizedEmail = email.trim().toLowerCase();
        const user = await this.authRepository.findUserByEmail(normalizedEmail);

        if (!user) {
            throw new HttpError(401, 'AUTH_INVALID_CREDENTIALS', 'Invalid credentials.');
        }

        const passwordMatches = await bcrypt.compare(password, user.password);

        if (!passwordMatches) {
            throw new HttpError(401, 'AUTH_INVALID_CREDENTIALS', 'Invalid credentials.');
        }

        const accessToken = jwt.sign({ email: user.email }, this.config.jwtAccessSecret, {
            expiresIn: ACCESS_TOKEN_TTL_SECONDS,
        });

        const refreshToken = jwt.sign({ email: user.email }, this.config.jwtRefreshSecret, {
            expiresIn: `${REFRESH_TOKEN_TTL_DAYS}d`,
        });

        await this.authRepository.saveRefreshToken(
            refreshToken,
            user.id,
            createExpiresAtFromNow(REFRESH_TOKEN_TTL_DAYS)
        );

        return { accessToken, refreshToken };
    }

    async refresh(refreshToken: string | undefined): Promise<RefreshResult> {
        if (!refreshToken) {
            throw new HttpError(403, 'AUTH_MISSING_REFRESH_TOKEN', 'Refresh token is missing.');
        }

        const storedToken = await this.authRepository.findRefreshToken(refreshToken);

        if (!storedToken) {
            throw new HttpError(403, 'AUTH_INVALID_REFRESH_TOKEN', 'Refresh token is invalid.');
        }

        if (storedToken.expiresAt < new Date()) {
            await this.authRepository.deleteRefreshToken(refreshToken);
            throw new HttpError(403, 'AUTH_EXPIRED_REFRESH_TOKEN', 'Refresh token is expired.');
        }

        try {
            jwt.verify(refreshToken, this.config.jwtRefreshSecret);
        } catch {
            await this.authRepository.deleteRefreshToken(refreshToken);
            throw new HttpError(403, 'AUTH_INVALID_REFRESH_TOKEN', 'Refresh token is invalid.');
        }

        const accessToken = jwt.sign({ email: storedToken.user.email }, this.config.jwtAccessSecret, {
            expiresIn: ACCESS_TOKEN_TTL_SECONDS,
        });

        return { accessToken };
    }

    async logout(refreshToken: string | undefined): Promise<void> {
        if (!refreshToken) {
            return;
        }

        await this.authRepository.deleteRefreshToken(refreshToken);
    }
}

