import {NextFunction, Request, Response} from "express";
import jwt, {JwtPayload} from "jsonwebtoken";


export interface AuthRequest extends Request {
    user?: string;
}

export function verifyToken(
    req: AuthRequest,
    res: Response,
    next: NextFunction
): void {
    const authHeader = req.headers.authorization;

    if (!authHeader) {
        res.sendStatus(401);
        return;
    }

    const token = authHeader.split(" ")[1];

    try {
        const decoded = jwt.verify(token, "SECRET") as JwtPayload;

        req.user = decoded.email as string;
        next();
    } catch {
        res.sendStatus(403);
    }
}

