import {NextFunction, Request, Response} from "express";
import jwt from "jsonwebtoken";


export interface AuthRequest extends Request {
    user?: string;
}

export function verifyToken(req: AuthRequest, res: Response, next: NextFunction) {
    const authHeader = req.headers.authorization;
    if (!authHeader) return res.sendStatus(401);

    const token = authHeader.split(" ")[1];

    jwt.verify(token, "SECRET", (err, user) => {
        if (err) return res.sendStatus(403);
        req.user = (user as any).email;
        next();
    });
}
