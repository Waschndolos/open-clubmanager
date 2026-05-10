import { HttpError } from './core/httpError.ts';

export interface ApiV2Config {
    jwtAccessSecret: string;
    jwtRefreshSecret: string;
    usesLegacyFallback: boolean;
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
    };
}

