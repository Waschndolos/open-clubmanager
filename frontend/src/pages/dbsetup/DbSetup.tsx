import React, { useState } from 'react';
import {
    Alert,
    Box,
    Button,
    Card,
    CardContent,
    CircularProgress,
    Divider,
    TextField,
    Typography,
} from '@mui/material';
import { Storage, FolderOpen } from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { saveDbPath } from '../../api/settings';
import { getSetupStatus } from '../../api/setup';

const isElectron = typeof window !== 'undefined' && !!window.electronDialog;

const DbSetup: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();

    const [manualPath, setManualPath] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    async function applyDbPath(dbPath: string) {
        setError(null);
        setLoading(true);
        try {
            await saveDbPath(dbPath);
            // Persist the path in Electron store so it is restored on next launch
            if (window.electronDialog) {
                window.electronDialog.storeDbPath(dbPath);
            }
            // Check if an admin user needs to be created
            const { setupRequired } = await getSetupStatus();
            if (setupRequired) {
                navigate('/setup', { replace: true });
            } else {
                navigate('/login', { replace: true });
            }
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('dbSetup.error.generic'));
        } finally {
            setLoading(false);
        }
    }

    async function handleSelectNewDirectory() {
        if (!window.electronDialog) return;
        const dir = await window.electronDialog.selectDbDirectory();
        if (!dir) return;
        const dbPath = `${dir}/clubmanager.db`;
        await applyDbPath(dbPath);
    }

    async function handleSelectExistingFile() {
        if (!window.electronDialog) return;
        const filePath = await window.electronDialog.selectDbFile();
        if (!filePath) return;
        await applyDbPath(filePath);
    }

    async function handleManualSubmit() {
        if (!manualPath.trim()) return;
        await applyDbPath(manualPath.trim());
    }

    return (
        <Box
            sx={{
                minHeight: '100vh',
                minWidth: '100vw',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                bgcolor: 'background.default',
                px: 2,
            }}
        >
            <Card sx={{ width: '100%', maxWidth: 520, p: 2 }} elevation={6}>
                <CardContent>
                    <Box display="flex" flexDirection="column" alignItems="center" mb={3}>
                        <Storage sx={{ fontSize: 48, color: 'primary.main', mb: 1 }} />
                        <Typography variant="h5" component="h1" gutterBottom align="center">
                            {t('dbSetup.title')}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" align="center">
                            {t('dbSetup.description')}
                        </Typography>
                    </Box>

                    {error && (
                        <Alert severity="error" variant="outlined" sx={{ mb: 2 }}>
                            {error}
                        </Alert>
                    )}

                    {loading && (
                        <Box display="flex" justifyContent="center" mb={2}>
                            <CircularProgress size={28} />
                        </Box>
                    )}

                    {isElectron ? (
                        <Box display="flex" flexDirection="column" gap={2}>
                            <Button
                                variant="contained"
                                fullWidth
                                startIcon={<FolderOpen />}
                                onClick={handleSelectNewDirectory}
                                disabled={loading}
                            >
                                {t('dbSetup.createNew')}
                            </Button>

                            <Divider>{t('dbSetup.or')}</Divider>

                            <Button
                                variant="outlined"
                                fullWidth
                                startIcon={<FolderOpen />}
                                onClick={handleSelectExistingFile}
                                disabled={loading}
                            >
                                {t('dbSetup.useExisting')}
                            </Button>
                        </Box>
                    ) : (
                        <Box display="flex" flexDirection="column" gap={2}>
                            <Typography variant="body2" color="text.secondary">
                                {t('dbSetup.manualPathLabel')}
                            </Typography>
                            <TextField
                                label={t('dbSetup.pathPlaceholder')}
                                variant="outlined"
                                fullWidth
                                value={manualPath}
                                onChange={(e) => setManualPath(e.target.value)}
                                placeholder="/path/to/clubmanager.db"
                                disabled={loading}
                            />
                            <Button
                                variant="contained"
                                fullWidth
                                onClick={handleManualSubmit}
                                disabled={!manualPath.trim() || loading}
                            >
                                {t('dbSetup.confirmPath')}
                            </Button>
                        </Box>
                    )}
                </CardContent>
            </Card>
        </Box>
    );
};

export default DbSetup;
