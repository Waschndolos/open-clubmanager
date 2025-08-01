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
    Typography
} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {apppreference, userpreference} from "../../lib/preferences";
import SettingsIcon from "@mui/icons-material/Settings";
import {validatePath} from "../../api/validation";
import {getDbPath, saveDbPath} from "../../api/settings";

export function Settings() {
    const {t, i18n} = useTranslation();

    const raw = apppreference.get('DATABASE_URL');
    const [dbPath, setDbPath] = useState<string>(typeof raw == 'string' ? raw : '');
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
        getDbPath().then((dbPath) => {
            setDbPath(dbPath)
        })

    }, []);

    const saveSettings = () => {
        userpreference?.set('1', 'language', language); // TODO: use userID as soon as we have auth
        i18n.changeLanguage(language);
        +apppreference.set('DATABASE_URL', dbPath);
        saveDbPath(dbPath).then(() => {
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
        return await validatePath(path);
    }

    return (
        <Box sx={{p: 4}}>
            <Typography variant="h4" gutterBottom>
                {t("settings.title")}
            </Typography>

            <Paper sx={{
                p: 3,
            }}>
                <SettingsIcon fontSize="large" color="primary"/>
                <FormControl fullWidth margin="normal">
                    <TextField
                        label={t("settings.labels.dbPath")}
                        variant="outlined"
                        value={dbPath}
                        fullWidth
                        error={dbPath.length === 0 || /[<>:"|?*]/.test(dbPath)}
                        helperText={dbPath.length === 0
                            ? t("settings.validatíon.error.required")
                            : /[<>:"|?*]/.test(dbPath)
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
                        placeholder="/path/to/database.db"
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
};
