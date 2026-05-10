import { NextFunction, Request, Response } from 'express';

export type AsyncRequestHandler = (
    req: Request,
    res: Response,
    next: NextFunction
) => Promise<void>;

export function asyncHandler(handler: AsyncRequestHandler) {
    return (req: Request, res: Response, next: NextFunction): void => {
        void handler(req, res, next).catch(next);
    };
}

