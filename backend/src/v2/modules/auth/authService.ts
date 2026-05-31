import bcrypt from 'bcrypt';
import jwt from 'jsonwebtoken';
import { randomBytes } from 'node:crypto';
import nodemailer from 'nodemailer';
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
const PASSWORD_RESET_TOKEN_TTL_HOURS = 1;

function createExpiresAtFromNow(days: number): Date {
    const result = new Date();
    result.setDate(result.getDate() + days);
    return result;
}

function createExpiresAtFromNowHours(hours: number): Date {
    const result = new Date();
    result.setTime(result.getTime() + hours * 60 * 60 * 1000);
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

    async requestPasswordReset(email: string): Promise<void> {
        if (!email) {
            throw new HttpError(400, 'AUTH_INVALID_INPUT', 'Email is required.');
        }

        const normalizedEmail = email.trim().toLowerCase();
        const user = await this.authRepository.findUserByEmail(normalizedEmail);

        // Always respond with success to avoid leaking whether an email exists
        if (!user) {
            return;
        }

        // Remove any existing reset tokens for this user
        await this.authRepository.deleteAllPasswordResetTokensForUser(user.id);

        const token = randomBytes(32).toString('hex');
        const expiresAt = createExpiresAtFromNowHours(PASSWORD_RESET_TOKEN_TTL_HOURS);

        await this.authRepository.savePasswordResetToken(token, user.id, expiresAt);

        const resetUrl = `${this.config.appBaseUrl}/reset-password?token=${token}`;

        if (this.config.smtpHost) {
            await this.sendPasswordResetEmail(user.email, resetUrl);
        } else {
            // Log the reset URL when SMTP is not configured (development fallback)
            console.log(`[Password Reset] Reset URL for ${user.email}: ${resetUrl}`);
        }
    }

    async resetPassword(token: string, newPassword: string): Promise<void> {
        if (!token || !newPassword) {
            throw new HttpError(400, 'AUTH_INVALID_INPUT', 'Token and new password are required.');
        }

        if (newPassword.length < 8) {
            throw new HttpError(400, 'AUTH_PASSWORD_TOO_SHORT', 'Password must be at least 8 characters long.');
        }

        const storedToken = await this.authRepository.findPasswordResetToken(token);

        if (!storedToken) {
            throw new HttpError(400, 'AUTH_INVALID_RESET_TOKEN', 'Password reset token is invalid.');
        }

        if (storedToken.expiresAt < new Date()) {
            await this.authRepository.deletePasswordResetToken(token);
            throw new HttpError(400, 'AUTH_EXPIRED_RESET_TOKEN', 'Password reset token has expired.');
        }

        const passwordHash = await bcrypt.hash(newPassword, 10);
        await this.authRepository.updateUserPassword(storedToken.userId, passwordHash);

        // Invalidate the used reset token and all refresh tokens for security
        await this.authRepository.deletePasswordResetToken(token);
        await this.authRepository.deleteAllRefreshTokensForUser(storedToken.userId);
    }

    private async sendPasswordResetEmail(toEmail: string, resetUrl: string): Promise<void> {
        const transporter = nodemailer.createTransport({
            host: this.config.smtpHost,
            port: this.config.smtpPort ?? 587,
            secure: (this.config.smtpPort ?? 587) === 465,
            auth: this.config.smtpUser
                ? {
                    user: this.config.smtpUser,
                    pass: this.config.smtpPassword,
                }
                : undefined,
        });

        await transporter.sendMail({
            from: this.config.smtpFrom ?? this.config.smtpUser ?? 'noreply@open-clubmanager',
            to: toEmail,
            subject: 'Password Reset – Open ClubManager',
            text: `You requested a password reset.\n\nClick the link below to set a new password (valid for ${PASSWORD_RESET_TOKEN_TTL_HOURS} hour):\n\n${resetUrl}\n\nIf you did not request this, please ignore this email.`,
            html: `<p>You requested a password reset.</p><p>Click the link below to set a new password (valid for ${PASSWORD_RESET_TOKEN_TTL_HOURS} hour):</p><p><a href="${resetUrl}">${resetUrl}</a></p><p>If you did not request this, please ignore this email.</p>`,
        });
    }
}
