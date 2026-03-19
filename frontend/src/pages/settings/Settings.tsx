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
    Typography,
    Chip,
} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {apppreference, userpreference} from "../../lib/preferences";
import SettingsIcon from "@mui/icons-material/Settings";
import FolderOpenIcon from "@mui/icons-material/FolderOpen";
import {validatePath} from "../../api/validation";
import {getDbPath, saveDbPath} from "../../api/settings";
import PageHeader from "../../components/common/PageHeader";
import {isElectronFolderMode} from "../../lib/environment";

export function Settings() {
    const {t, i18n} = useTranslation();
    const folderMode = isElectronFolderMode();

    const raw = folderMode ? null : apppreference.get('DATABASE_URL');
    const [dbPath, setDbPath] = useState<string>(typeof raw == 'string' ? raw : '');
    const [clubFolder, setClubFolder] = useState<string>('');
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
        if (folderMode) {
            window.api!.club.getFolder().then((folder) => {
                setClubFolder(folder ?? '');
            }).catch(() => {/* ignore */});
        } else {
            getDbPath().then((path) => {
                setDbPath(path);
            }).catch(() => {/* ignore when backend not available */});
        }
    }, [folderMode]);

    const handleChangeFolder = async () => {
        try {
            const folder = await window.api!.club.selectFolder();
            if (folder) {
                setClubFolder(folder);
                setSnackBarState({ open: true, message: t("settings.labels.saveSuccess") });
            }
        } catch {
            setSnackBarState({ open: true, message: t("settings.labels.saveFails") });
        }
    };

    const saveSettings = () => {
        userpreference?.set('1', 'language', language);
        i18n.changeLanguage(language);
        if (!folderMode) {
            apppreference.set('DATABASE_URL', dbPath);
            saveDbPath(dbPath).then(() => {
                setSnackBarState({
                    open: true,
                    message: t("settings.labels.saveSuccess")
                });
            }, () => {
                setSnackBarState({
                    open: true,
                    message: t("settings.labels.saveFails")
                });
            });
        } else {
            setSnackBarState({ open: true, message: t("settings.labels.saveSuccess") });
        }
    };

    async function validateDbPath(path: string): Promise<{ valid: boolean, i18nToken?: string }> {
        return await validatePath(path);
    }

    return (
        <Box sx={{p: 4}}>
            <PageHeader
                title={t("settings.title")}
                icon={<SettingsIcon fontSize="small" />}
            />

            <Paper sx={{ p: 3 }}>
                {folderMode ? (
                    <FormControl fullWidth margin="normal">
                        <Box display="flex" alignItems="center" gap={2} flexWrap="wrap">
                            <Typography variant="body2" color="text.secondary" sx={{ minWidth: 120 }}>
                                {t("settings.labels.clubFolder")}
                            </Typography>
                            <Chip
                                icon={<FolderOpenIcon />}
                                label={clubFolder || t("settings.labels.noFolderSelected")}
                                variant="outlined"
                                sx={{ fontFamily: 'monospace', maxWidth: 400, overflow: 'hidden' }}
                            />
                            <Button
                                variant="outlined"
                                size="small"
                                startIcon={<FolderOpenIcon />}
                                onClick={handleChangeFolder}
                            >
                                {t("settings.labels.changeFolder")}
                            </Button>
                        </Box>
                    </FormControl>
                ) : (
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
                                setValidationMessage("");
                            }}
                            onBlur={async () => {
                                const response = await validateDbPath(dbPath).catch(() => ({ valid: false, i18nToken: 'error.required' }));
                                setValidationMessage(
                                    response.valid
                                        ? t(`settings.validatíon.${response.i18nToken}`)
                                        : t(`settings.validatíon.${response.i18nToken}`)
                                );
                            }}
                            placeholder="/path/to/database.db"
                            margin="normal"/>
                    </FormControl>
                )}

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
