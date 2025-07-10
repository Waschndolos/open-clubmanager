import fs from 'fs';
import express from 'express';

const router = express.Router();

router.post('/check-db-path', async (req, res) => {
    const {path} = req.body;
    console.log("Received path:", path);
    let responseBody = {valid: true, error: ''};
    if (!path) {
        responseBody = {valid: false, error: 'error.required'};
        res.status(400).json(responseBody);
    } else {
        fs.stat(path, () => {

            const checkSqlite = (filePath: string, callback: (isSqlite: boolean) => void) => {
                console.log("Checking if path is a SQLite database...");
                fs.open(filePath, 'r', (err, fd) => {
                    if (err) return callback(false);
                    const buffer = Buffer.alloc(16);
                    fs.read(fd, buffer, 0, 16, 0, (err) => {
                        fs.close(fd, () => {
                        });
                        if (err) return callback(false);
                        // SQLite-Header: "SQLite format 3\0"
                        const header = buffer.toString('utf8', 0, 16);
                        callback(header.startsWith('SQLite format 3'));
                    });
                });
            };

            fs.access(path, fs.constants.F_OK, (err) => {
                if (err) {
                    console.log("File does not exist, trying to create it.");
                    // File does not exist, try to create it
                    return fs.open(path, 'w', (err) => {
                        if (err) {
                            console.log("Error creating file:", err);
                            responseBody = {valid: false, error: 'error.cannotcreate'};
                            return res.json(responseBody);
                        } else {
                            console.log("File created successfully.");
                            responseBody = {valid: true, error: ''};
                            return res.json(responseBody);
                        }
                    });
                } else {
                    // Datei exists, check if it's a SQLite DB
                    return checkSqlite(path, (isSqlite) => {
                        if (!isSqlite) {
                            console.log("File is not a SQLite database.");
                            responseBody = {valid: false, error: 'error.nosqllitedb'};
                            return res.json(responseBody);
                        } else {
                            responseBody = {valid: true, error: ''};
                            return res.json(responseBody);
                        }
                    });
                }
            });
        });
    }
});

export default router;