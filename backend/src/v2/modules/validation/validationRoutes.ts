import fs from 'fs';
import { Router } from 'express';
import { DatabaseMode, isDatabaseUrlValid } from '../preferences/appConfigStore.ts';

export function createValidationRoutes(): Router {
    const router = Router();

    router.post('/check-db-path', async (req, res) => {
        const { path } = req.body as { path?: string };

        if (!path) {
            res.status(400).json({ valid: false, i18nToken: 'error.required' });
            return;
        }

        const checkSqlite = (filePath: string, callback: (isSqlite: boolean) => void) => {
            fs.open(filePath, 'r', (openErr, fd) => {
                if (openErr) {
                    callback(false);
                    return;
                }

                const buffer = Buffer.alloc(16);
                fs.read(fd, buffer, 0, 16, 0, (readErr) => {
                    fs.close(fd, () => undefined);
                    if (readErr) {
                        callback(false);
                        return;
                    }
                    const header = buffer.toString('utf8', 0, 16);
                    callback(header.startsWith('SQLite format 3'));
                });
            });
        };

        fs.access(path, fs.constants.F_OK, (accessErr) => {
            if (accessErr) {
                fs.open(path, 'w', (createErr) => {
                    if (createErr) {
                        res.json({ valid: false, i18nToken: 'error.cannotcreate' });
                        return;
                    }

                    fs.unlink(path, () => {
                        res.json({ valid: true, i18nToken: 'success.newdb' });
                    });
                });
                return;
            }

            checkSqlite(path, (isSqlite) => {
                if (!isSqlite) {
                    res.json({ valid: false, i18nToken: 'error.nosqllitedb' });
                    return;
                }
                res.json({ valid: true, i18nToken: 'success.valid' });
            });
        });
    });

    router.post('/check-db-url', async (req, res) => {
        const { mode, databaseUrl } = req.body as { mode?: DatabaseMode; databaseUrl?: string };

        if (mode !== 'sqlite-local' && mode !== 'mysql-shared') {
            res.status(400).json({ valid: false, i18nToken: 'error.invalidmode' });
            return;
        }

        const normalizedUrl = (databaseUrl ?? '').trim();
        if (!isDatabaseUrlValid(mode, normalizedUrl)) {
            res.json({
                valid: false,
                i18nToken: mode === 'sqlite-local' ? 'error.invalidsqliteurl' : 'error.invalidmysqlurl',
            });
            return;
        }

        res.json({ valid: true, i18nToken: 'success.valid' });
    });

    return router;
}

