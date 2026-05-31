import { Response, Router } from 'express';
import { asyncHandler } from '../../core/asyncHandler.ts';
import { createVerifyToken } from './authMiddleware.ts';
import { ApiV2Config } from '../../config.ts';
import { AuthService } from './authService.ts';
import { HttpError } from '../../core/httpError.ts';
import { AuthenticatedRequest } from './authMiddleware.ts';

const REFRESH_COOKIE_NAME = 'refreshToken';
const REFRESH_COOKIE_PATH = '/api/v2/auth/refresh-token';

export function createAuthRoutes(config: ApiV2Config, authService: AuthService): Router {
    const router = Router();
    const verifyToken = createVerifyToken(config);

    const writeRefreshCookie = (res: Response, refreshToken: string): void => {
        res.cookie(REFRESH_COOKIE_NAME, refreshToken, {
            httpOnly: true,
            secure: process.env.NODE_ENV === 'production',
            sameSite: 'strict',
            path: REFRESH_COOKIE_PATH,
            maxAge: 7 * 24 * 60 * 60 * 1000,
        });
    };

    const clearRefreshCookie = (res: Response): void => {
        res.clearCookie(REFRESH_COOKIE_NAME, {
            path: REFRESH_COOKIE_PATH,
        });
    };

    router.post('/login', asyncHandler(async (req, res) => {
        const { email, password } = req.body as { email?: string; password?: string };
        const result = await authService.login(email ?? '', password ?? '');
        writeRefreshCookie(res, result.refreshToken);
        res.json({ accessToken: result.accessToken });
    }));

    router.post('/refresh-token', asyncHandler(async (req, res) => {
        const refreshToken = req.cookies?.[REFRESH_COOKIE_NAME] as string | undefined;
        const result = await authService.refresh(refreshToken);
        res.json(result);
    }));

    router.post('/logout', asyncHandler(async (req, res) => {
        const refreshToken = req.cookies?.[REFRESH_COOKIE_NAME] as string | undefined;
        await authService.logout(refreshToken);
        clearRefreshCookie(res);
        res.sendStatus(204);
    }));

    router.get('/profile', verifyToken, (req, res) => {
        const authReq = req as AuthenticatedRequest;
        if (!authReq.userEmail) {
            throw new HttpError(401, 'AUTH_INVALID_TOKEN', 'User identity is missing in token.');
        }
        res.json({ email: authReq.userEmail });
    });

    router.post('/forgot-password', asyncHandler(async (req, res) => {
        const { email } = req.body as { email?: string };
        await authService.requestPasswordReset(email ?? '');
        // Always return 200 to avoid leaking whether the email exists
        res.json({ message: 'If an account with that email exists, a password reset link has been sent.' });
    }));

    router.post('/reset-password', asyncHandler(async (req, res) => {
        const { token, newPassword } = req.body as { token?: string; newPassword?: string };
        await authService.resetPassword(token ?? '', newPassword ?? '');
        res.json({ message: 'Password has been reset successfully.' });
    }));

    return router;
}

