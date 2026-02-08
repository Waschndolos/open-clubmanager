import { Router } from "express";
import jwt from "jsonwebtoken";
import bcrypt from "bcrypt";
import { verifyToken } from "../middlewares/auth";
import { getClient } from "../db.ts";

const router = Router();

const ACCESS_SECRET = "SECRET";
const REFRESH_SECRET = "REFRESH_SECRET";

router.post("/login", async (req, res) => {
    const { email, password } = req.body;

    const prisma = await getClient();
    const user = await prisma.user.findUnique({ where: { email } });

    if (!user || !(await bcrypt.compare(password, user.password))) {
        return res.status(401).json({ error: "Invalid credentials" });
    }

    const accessToken = jwt.sign({ email }, ACCESS_SECRET, { expiresIn: "1m" });
    const refreshToken = jwt.sign({ email }, REFRESH_SECRET, { expiresIn: "7d" });

    // Store refresh token in the database
    await prisma.refreshToken.create({
        data: {
            token: refreshToken,
            userId: user.id,
            expiresAt: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days
        },
    });

    // Set httpOnly cookie for refresh token
    res.cookie("refreshToken", refreshToken, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "strict",
        path: "/auth/refresh-token",
        maxAge: 7 * 24 * 60 * 60 * 1000,
    });

    res.json({ accessToken });
});

router.post("/refresh-token", async (req, res) => {
    const refreshToken = req.cookies.refreshToken;
    if (!refreshToken) return res.sendStatus(403);

    const prisma = await getClient();

    const storedToken = await prisma.refreshToken.findUnique({
        where: { token: refreshToken },
        include: { user: true },
    });

    if (!storedToken) return res.sendStatus(403);
    if (storedToken.expiresAt < new Date()) {
        // Token expired, delete it
        await prisma.refreshToken.delete({ where: { token: refreshToken } });
        return res.sendStatus(403);
    }

    jwt.verify(refreshToken, REFRESH_SECRET, (err, user) => {
        if (err) return res.sendStatus(403);

        const accessToken = jwt.sign({ email: (user as any).email }, ACCESS_SECRET, {
            expiresIn: "15m",
        });
        res.json({ accessToken });
    });
});

router.post("/logout", async (req, res) => {
    const refreshToken = req.cookies.refreshToken;
    if (!refreshToken) return res.sendStatus(204);

    const prisma = await getClient();

    await prisma.refreshToken.deleteMany({ where: { token: refreshToken } });

    res.clearCookie("refreshToken", { path: "/auth/refresh-token" });
    res.sendStatus(204);
});

router.get("/profile", verifyToken, (req, res) => {
    res.json({ email: (req as any).user });
});

export default router;
