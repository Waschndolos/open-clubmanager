import React, {useEffect, useRef, useState} from 'react';
import {
    Alert,
    Box,
    Button,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    Fade,
    FormControl,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    Snackbar,
    TextField,
    Typography,
} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {apppreference, userpreference} from "../../lib/preferences";
import SettingsIcon from "@mui/icons-material/Settings";
import {DatabaseMode, getDatabaseSettings, saveDatabaseSettings} from "../../api/settings";
import {validatePath} from "../../api/validation";
import {createBackup, restoreBackup} from "../../api/backup";
import PageHeader from "../../components/common/PageHeader";

const LAST_BACKUP_KEY = 'lastBackupDate';

export function Settings() {
    const {t, i18n} = useTranslation();

    const raw = apppreference.get('DATABASE_URL');
    const [dbPath, setDbPath] = useState<string>(typeof raw == 'string' ? raw : '');
    const [databaseMode, setDatabaseMode] = useState<DatabaseMode>('sqlite-local');
    const [language, setLanguage] = useState<string>('en');
    const [validationMessage, setValidationMessage] = useState<string>("");
    const [snackBarState, setSnackBarState] = useState<{ open: boolean, message: string }>({
        open: false,
        message: ""
    });

    // Backup / restore state
    const [backupLoading, setBackupLoading] = useState(false);
    const [restoreLoading, setRestoreLoading] = useState(false);
    const [lastBackupDate, setLastBackupDate] = useState<string | null>(
        localStorage.getItem(LAST_BACKUP_KEY)
    );
    const [pendingRestoreFile, setPendingRestoreFile] = useState<File | null>(null);
    const [confirmOpen, setConfirmOpen] = useState(false);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const availableLanguages = () => {
        return Object.keys(i18n.services.resourceStore.data);
    }

    useEffect(() => {
        const raw = userpreference?.get('1', 'language');
        const savedLanguage = typeof raw === 'string' ? raw : 'de';

        setLanguage(savedLanguage);
        i18n.changeLanguage(savedLanguage);
    }, [i18n]);

    useEffect(() => {
        getDatabaseSettings().then(({ mode, databaseUrl }) => {
            setDatabaseMode(mode);
            setDbPath(databaseUrl);
        });

    }, []);

    const saveSettings = () => {
        userpreference?.set('1', 'language', language); // TODO: use userID as soon as we have auth
        i18n.changeLanguage(language);
        apppreference.set('DATABASE_URL', dbPath);
        apppreference.set('DATABASE_MODE', databaseMode);
        saveDatabaseSettings({ mode: databaseMode, databaseUrl: dbPath }).then(() => {
            setSnackBarState({
                open: true,
                message: t("settings.labels.saveSuccess")
            });
        }, () => {
            setSnackBarState({
                open: true,
                message: t("settings.labels.saveFails")
            })
        });

    };

    async function validateDbPath(path: string): Promise<{ valid: boolean, i18nToken?: string }> {
        if (databaseMode === 'mysql-shared') {
            const valid = /^mysqls?:\/\//i.test(path);
            return {
                valid,
                i18nToken: valid ? 'success.valid' : 'error.invalidmysqlurl',
            };
        }

        return await validatePath(path.startsWith('file:') ? path.replace('file:', '').replace(/^\/\//, '') : path);
    }

    const handleBackup = async () => {
        setBackupLoading(true);
        try {
            await createBackup();
            const now = new Date().toISOString();
            localStorage.setItem(LAST_BACKUP_KEY, now);
            setLastBackupDate(now);
            setSnackBarState({ open: true, message: t("settings.backup.backupSuccess") });
        } catch {
            setSnackBarState({ open: true, message: t("settings.backup.backupError") });
        } finally {
            setBackupLoading(false);
        }
    };

    const handleRestoreFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (file) {
            setPendingRestoreFile(file);
            setConfirmOpen(true);
        }
        // Reset input so the same file can be re-selected after a cancel
        if (fileInputRef.current) {
            fileInputRef.current.value = '';
        }
    };

    const handleRestoreConfirm = async () => {
        if (!pendingRestoreFile) return;
        setConfirmOpen(false);
        setRestoreLoading(true);
        try {
            await restoreBackup(pendingRestoreFile);
            setSnackBarState({ open: true, message: t("settings.backup.restoreSuccess") });
        } catch {
            setSnackBarState({ open: true, message: t("settings.backup.restoreError") });
        } finally {
            setRestoreLoading(false);
            setPendingRestoreFile(null);
        }
    };

    const formattedLastBackup = lastBackupDate
        ? new Date(lastBackupDate).toLocaleString()
        : t("settings.backup.never");

    return (
        <Box sx={{p: 4}}>
            <PageHeader
                title={t("settings.title")}
                icon={<SettingsIcon fontSize="small" />}
            />

            <Paper sx={{
                p: 3,
            }}>
                <FormControl fullWidth margin="normal">
                    <InputLabel id="database-mode-select-label">{t("settings.labels.databaseMode")}</InputLabel>
                    <Select
                        labelId="database-mode-select-label"
                        value={databaseMode}
                        label={t("settings.labels.databaseMode")}
                        onChange={(e) => {
                            const mode = e.target.value as DatabaseMode;
                            setDatabaseMode(mode);
                            if (mode === 'sqlite-local' && dbPath.length === 0) {
                                setDbPath('file:./clubmanager.db');
                            }
                        }}
                    >
                        <MenuItem value="sqlite-local">{t("settings.databaseModes.sqliteLocal")}</MenuItem>
                        <MenuItem value="mysql-shared">{t("settings.databaseModes.mysqlShared")}</MenuItem>
                    </Select>
                </FormControl>

                <FormControl fullWidth margin="normal">
                    <TextField
                        label={databaseMode === 'sqlite-local' ? t("settings.labels.dbPath") : t("settings.labels.dbUrl")}
                        variant="outlined"
                        value={dbPath}
                        fullWidth
                        error={dbPath.length === 0}
                        helperText={dbPath.length === 0
                            ? t("settings.validatíon.error.required")
                            : validationMessage}
                        onChange={(e) => {
                            setDbPath(e.target.value);
                            setValidationMessage(""); // Reset message on input change
                        }}
                        onBlur={async () => {
                            const response = await validateDbPath(dbPath);
                            setValidationMessage(
                                response.valid
                                    ? t(`settings.validatíon.${response.i18nToken}`)
                                    : t(`settings.validatíon.${response.i18nToken}`)
                            );
                        }}
                        placeholder={databaseMode === 'sqlite-local' ? "file:/path/to/database.db" : "******host:3306/db"}
                        margin="normal"/>
                </FormControl>

                <FormControl fullWidth margin="normal">
                    <InputLabel id="language-select-label">{t("settings.labels.language")}</InputLabel>
                    <Select
                        labelId="language-select-label"
                        value={language}
                        label={t("settings.labels.language")}
                        onChange={(e) => {
                            const lng = e.target.value;
                            setLanguage(lng);
                        }}
                    >
                        {availableLanguages().map((lng) => (
                            <MenuItem key={lng} value={lng}>
                                {t(`settings.languages.${lng}`) || lng}
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </Paper>

            <Button
                variant="contained"
                color="primary"
                onClick={saveSettings}
                sx={{mt: 2}}
            >
                {t("buttons.save")}
            </Button>

            {/* ── Backup & Restore ─────────────────────────────────────────── */}
            <Typography variant="h6" sx={{mt: 4, mb: 1}}>
                {t("settings.backup.title")}
            </Typography>

            {databaseMode === 'mysql-shared' ? (
                <Alert severity="info" sx={{mb: 2}}>
                    {t("settings.backup.mysqlHint")}
                </Alert>
            ) : (
                <Paper sx={{p: 3}}>
                    <Typography variant="body2" color="text.secondary" sx={{mb: 2}}>
                        {t("settings.backup.description")}
                    </Typography>

                    <Typography variant="body2" sx={{mb: 2}}>
                        {t("settings.backup.lastBackup")}: <strong>{formattedLastBackup}</strong>
                    </Typography>

                    <Button
                        variant="contained"
                        color="primary"
                        onClick={handleBackup}
                        disabled={backupLoading}
                        startIcon={backupLoading ? <CircularProgress size={16} /> : undefined}
                        sx={{mr: 2, mb: 2}}
                    >
                        {t("settings.backup.backupButton")}
                    </Button>

                    <Typography variant="subtitle2" sx={{mt: 2, mb: 1}}>
                        {t("settings.backup.restoreTitle")}
                    </Typography>
                    <Typography variant="body2" color="text.secondary" sx={{mb: 1}}>
                        {t("settings.backup.restoreDescription")}
                    </Typography>

                    <input
                        ref={fileInputRef}
                        type="file"
                        accept=".zip"
                        style={{display: 'none'}}
                        onChange={handleRestoreFileChange}
                    />
                    <Button
                        variant="outlined"
                        color="warning"
                        onClick={() => fileInputRef.current?.click()}
                        disabled={restoreLoading}
                        startIcon={restoreLoading ? <CircularProgress size={16} /> : undefined}
                    >
                        {t("settings.backup.selectFile")}
                    </Button>
                </Paper>
            )}

            <Snackbar
                open={snackBarState.open}
                color="primary"
                onClose={() => setSnackBarState({...snackBarState, open: false})}
                slots={{
                    transition: Fade
                }}
                message={snackBarState.message}
                autoHideDuration={3000}/>

            {/* ── Restore confirmation dialog ──────────────────────────────── */}
            <Dialog open={confirmOpen} onClose={() => { setConfirmOpen(false); setPendingRestoreFile(null); }}>
                <DialogTitle>{t("settings.backup.confirmTitle")}</DialogTitle>
                <DialogContent>
                    <DialogContentText>
                        {t("settings.backup.confirmDescription", { filename: pendingRestoreFile?.name ?? '' })}
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => { setConfirmOpen(false); setPendingRestoreFile(null); }}>
                        {t("buttons.abort")}
                    </Button>
                    <Button onClick={handleRestoreConfirm} color="error" variant="contained">
                        {t("settings.backup.confirmButton")}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    )
}
