import {Router} from "express";
import jwt from "jsonwebtoken";
import bcrypt from "bcrypt";
import {verifyToken} from "../middlewares/auth";
import {getClient} from "../db.ts";

const router = Router();

const ACCESS_SECRET = "SECRET";
const REFRESH_SECRET = "REFRESH_SECRET";
const refreshTokens: string[] = [];

// router.post("/register", async (req, res) => {
//     const { email, password } = req.body;
//     const hashed = await bcrypt.hash(password, 10);
//
//     try {
//         await db.run("INSERT INTO users (email, password) VALUES (?, ?)", email, hashed);
//         res.sendStatus(201);
//     } catch {
//         res.status(400).json({ error: "User exists" });
//     }
// });

router.post("/login", async (req, res) => {
    const {email, password} = req.body;

    const prisma = await getClient();
    const user = await prisma.user.findUnique({where: {email: email}})

    console.log("User found:", user);


    if (!user || !(await bcrypt.compare(password, user.password))) {
        return res.status(401).json({error: "Invalid credentials"});
    }

    const accessToken = jwt.sign({email}, ACCESS_SECRET, {expiresIn: "15m"});
    const refreshToken = jwt.sign({email}, REFRESH_SECRET, {expiresIn: "7d"});

    refreshTokens.push(refreshToken);

    res.json({accessToken, refreshToken});
});

router.post("/refresh-token", (req, res) => {
    const {refreshToken} = req.body;

    if (!refreshToken || !refreshTokens.includes(refreshToken)) {
        return res.sendStatus(403);
    }

    jwt.verify(refreshToken, REFRESH_SECRET, (err, user) => {
        if (err) return res.sendStatus(403);

        const accessToken = jwt.sign({email: (user as any).email}, ACCESS_SECRET, {
            expiresIn: "15m"
        });
        res.json({accessToken});
    });
});

router.get("/profile", verifyToken, (req, res) => {
    res.json({email: (req as any).user});
});

export default router;
