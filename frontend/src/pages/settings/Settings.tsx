import React, {useEffect, useState} from 'react';
import {
    Box,
    Button,
    Fade,
    FormControl,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    Snackbar,
    TextField,
} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {apppreference, userpreference} from "../../lib/preferences";
import SettingsIcon from "@mui/icons-material/Settings";
import {DatabaseMode, getDatabaseSettings, saveDatabaseSettings} from "../../api/settings";
import {validatePath} from "../../api/validation";
import PageHeader from "../../components/common/PageHeader";

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
                        placeholder={databaseMode === 'sqlite-local' ? "file:/path/to/database.db" : "mysql://user:pass@host:3306/db"}
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
            <Snackbar
                open={snackBarState.open}
                color="primary"
                onClose={() => setSnackBarState({...snackBarState, open: false})}
                slots={{
                    transition: Fade
                }}
                message={snackBarState.message}
                autoHideDuration={3000}/>

        </Box>
    )
}
