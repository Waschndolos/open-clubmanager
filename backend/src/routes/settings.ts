import express from 'express';
import {getCurrentDbPath, setActiveDb} from "../db.ts";


const router = express.Router();


router.post("/set-db-path", async (req, res) => {
    const {dbPath} = req.body;
    try {
        await setActiveDb(dbPath);
        res.json({status: "ok", dbPath});
    } catch (err) {
        console.error(err);
        res.status(500).json({error: "Failed to set database path"});
    }
});

router.get('/db-path', async (req, res) => {
    try {
        const dbPath = getCurrentDbPath();
        res.json({dbPath});
    } catch (err) {
        console.error(err);
        res.status(500).json({error: "Failed to get database path"});
    }
})

export default router;