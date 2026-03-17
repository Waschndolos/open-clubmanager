import {Request, Response, Router} from "express";
import bcrypt from "bcrypt";
import {getClient} from "../db.ts";

const router = Router();

/**
 * GET /api/setup/status
 * Returns whether initial setup is required (no users exist yet).
 */
router.get("/status", async (_req: Request, res: Response): Promise<void> => {
    const prisma = await getClient();
    const userCount = await prisma.user.count();
    res.json({ setupRequired: userCount === 0 });
});

/**
 * POST /api/setup/initialize
 * Creates the initial admin user. Only allowed when no users exist.
 */
router.post("/initialize", async (req: Request, res: Response): Promise<void> => {
    const prisma = await getClient();
    const userCount = await prisma.user.count();

    if (userCount > 0) {
        res.status(403).json({ error: "Setup already completed." });
        return;
    }

    const { email, password } = req.body;

    if (!email || !password) {
        res.status(400).json({ error: "Email and password are required." });
        return;
    }

    if (typeof email !== "string" || typeof password !== "string") {
        res.status(400).json({ error: "Invalid input." });
        return;
    }

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    if (!emailRegex.test(email)) {
        res.status(400).json({ error: "Invalid email address." });
        return;
    }

    if (password.length < 8) {
        res.status(400).json({ error: "Password must be at least 8 characters long." });
        return;
    }

    // Ensure default roles exist
    const existingRoles = await prisma.role.count();
    if (existingRoles === 0) {
        await prisma.role.createMany({
            data: [
                { name: "Admin", id: 0 },
                { name: "Trainer", id: 1 },
                { name: "Member", id: 2 },
            ],
        });
    }

    const hashedPassword = await bcrypt.hash(password, 10);

    await prisma.user.create({
        data: {
            email,
            password: hashedPassword,
            roles: {
                connect: { id: 0 },
            },
        },
    });

    res.status(201).json({ message: "Admin user created successfully." });
});

export default router;
