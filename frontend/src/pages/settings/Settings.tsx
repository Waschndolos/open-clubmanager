import React, {useState, useEffect} from 'react';
import {
    Typography,
    TextField,
    Select,
    MenuItem,
    Button,
    FormControl,
    InputLabel,
    Paper,
    Fade,
    Box,
    Snackbar
} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {apppreference, userpreference} from "../../lib/preferences";
import GroupsIcon from "@mui/icons-material/Groups";
import SettingsIcon from "@mui/icons-material/Settings";

export function Settings() {
    const {t, i18n} = useTranslation();

    const raw = apppreference.get('DATABASE_URL');
    const [dbPath, setDbPath] = useState<string>(typeof raw == 'string' ? raw : '');
    const [language, setLanguage] = useState<string>('en');
    const [snackBarState, setsnackBarState] = useState<{ open: boolean, message: string }>({
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

    const saveSettings = () => {
        userpreference?.set('1', 'language', language); // TODO: use userID as soon as we have auth
        i18n.changeLanguage(language);
        +apppreference.set('DATABASE_URL', dbPath)

        setsnackBarState({
            open: true,
            message: t("settings.labels.saveSuccess")
        });
    };

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
                        onChange={(e) => setDbPath(e.target.value)}
                        placeholder="/path/to/database.db"
                        margin="normal"
                    />
                </FormControl>

                <FormControl fullWidth margin="normal">
                    <InputLabel id="language-select-label">{t("settings.labels.dbPath")}</InputLabel>
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
                onClose={() => setsnackBarState({...snackBarState, open: false})}
                slots={{
                    transition: Fade
                }}
                message={snackBarState.message}
                autoHideDuration={3000}
            />

        </Box>
    );
};
