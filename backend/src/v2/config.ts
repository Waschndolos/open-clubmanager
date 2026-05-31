import { HttpError } from './core/httpError.ts';

export interface ApiV2Config {
    jwtAccessSecret: string;
    jwtRefreshSecret: string;
    usesLegacyFallback: boolean;
    smtpHost?: string;
    smtpPort?: number;
    smtpUser?: string;
    smtpPassword?: string;
    smtpFrom?: string;
    appBaseUrl?: string;
}

/**
 * Reads and validates environment variables used by the v2 backend module.
 */
export function readApiV2Config(env: NodeJS.ProcessEnv = process.env): ApiV2Config {
    const jwtAccessSecret = env.JWT_ACCESS_SECRET ?? 'SECRET';
    const jwtRefreshSecret = env.JWT_REFRESH_SECRET ?? 'REFRESH_SECRET';
    const usesLegacyFallback = !env.JWT_ACCESS_SECRET || !env.JWT_REFRESH_SECRET;

    if (!jwtAccessSecret || !jwtRefreshSecret) {
        throw new HttpError(500, 'CONFIG_MISSING', 'JWT secret configuration is invalid.');
    }

    return {
        jwtAccessSecret,
        jwtRefreshSecret,
        usesLegacyFallback,
        smtpHost: env.SMTP_HOST,
        smtpPort: env.SMTP_PORT ? parseInt(env.SMTP_PORT, 10) : undefined,
        smtpUser: env.SMTP_USER,
        smtpPassword: env.SMTP_PASSWORD,
        smtpFrom: env.SMTP_FROM,
        appBaseUrl: env.APP_BASE_URL ?? 'http://localhost:5173',
    };
}

