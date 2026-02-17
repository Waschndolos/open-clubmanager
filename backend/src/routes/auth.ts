import {Request, Response, Router} from "express";
import jwt, {JwtPayload} from "jsonwebtoken";
import bcrypt from "bcrypt";
import {AuthRequest, verifyToken} from "../middlewares/auth";
import {getClient} from "../db.ts";

const router = Router();

const ACCESS_SECRET = "SECRET";
const REFRESH_SECRET = "REFRESH_SECRET";

/**
 * LOGIN
 */
router.post("/login", async (req: Request, res: Response): Promise<void> => {
    const { email, password } = req.body;

    const prisma = await getClient();
    const user = await prisma.user.findUnique({ where: { email } });

    if (!user || !(await bcrypt.compare(password, user.password))) {
        res.status(401).json({ error: "Invalid credentials" });
        return;
    }

    const accessToken = jwt.sign({ email }, ACCESS_SECRET, { expiresIn: "1m" });
    const refreshToken = jwt.sign({ email }, REFRESH_SECRET, { expiresIn: "7d" });

    await prisma.refreshToken.create({
        data: {
            token: refreshToken,
            userId: user.id,
            expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
        },
    });

    res.cookie("refreshToken", refreshToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "strict",
        path: "/auth/refresh-token",
        maxAge: 7 * 24 * 60 * 60 * 1000,
    });

    res.json({ accessToken });
});

/**
 * REFRESH TOKEN
 */
router.post("/refresh-token", async (req: Request, res: Response) : Promise<void> => {
    const refreshToken = req.cookies?.refreshToken;

    if (!refreshToken) {
        res.sendStatus(403);
        return;
    }

    const prisma = await getClient();

    const storedToken = await prisma.refreshToken.findUnique({
        where: { token: refreshToken },
        include: { user: true },
    });

    if (!storedToken) {
        res.sendStatus(403);
        return;
    }

    if (storedToken.expiresAt < new Date()) {
        await prisma.refreshToken.delete({ where: { token: refreshToken } });
        res.sendStatus(403);
        return;
    }

    try {
        const decoded = jwt.verify(refreshToken, REFRESH_SECRET) as JwtPayload;

        const accessToken = jwt.sign(
            { email: decoded.email },
            ACCESS_SECRET,
            { expiresIn: "15m" }
        );

        res.json({ accessToken });
    } catch {
        res.sendStatus(403);
    }
    });

/**
 * LOGOUT
 */
router.post("/logout", async (req: Request, res: Response): Promise<void> => {
    const refreshToken = req.cookies?.refreshToken;

    if (!refreshToken) {
        res.sendStatus(204);
        return;
    }

    const prisma = await getClient();

    await prisma.refreshToken.deleteMany({
        where: { token: refreshToken },
    });

    res.clearCookie("refreshToken", {
        path: "/auth/refresh-token",
    });

    res.sendStatus(204);
});

router.get("/profile", verifyToken, (req, res) => {
    res.json({ email: (req as AuthRequest).user });
});

export default router;
