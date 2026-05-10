import { NextFunction, Request, Response } from 'express';
import jwt, { JwtPayload } from 'jsonwebtoken';
import { ApiV2Config } from '../../config.ts';

export interface AuthenticatedRequest extends Request {
    userEmail?: string;
}

export function createVerifyToken(config: ApiV2Config) {
    return (req: AuthenticatedRequest, res: Response, next: NextFunction): void => {
        const authHeader = req.headers.authorization;

        if (!authHeader?.startsWith('Bearer ')) {
            res.status(401).json({ error: 'Missing bearer token.' });
            return;
        }

        const token = authHeader.slice('Bearer '.length);

        try {
            const decoded = jwt.verify(token, config.jwtAccessSecret) as JwtPayload;
            req.userEmail = typeof decoded.email === 'string' ? decoded.email : undefined;

            if (!req.userEmail) {
                res.status(401).json({ error: 'Token payload is invalid.' });
                return;
            }

            next();
        } catch {
            res.status(403).json({ error: 'Token is invalid or expired.' });
        }
    };
}

