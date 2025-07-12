import express from 'express';
import {setActiveDb} from "../db.ts";


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

export default router;