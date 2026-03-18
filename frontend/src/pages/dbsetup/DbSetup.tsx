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

declare global {
    interface Window {
        electronDialog?: {
            selectNewDbPath(): Promise<string | null>;
            selectExistingDbPath(): Promise<string | null>;
        };
    }
}

const DbSetup: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();

    const [manualPath, setManualPath] = useState('');
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);

    const isElectron = !!window.electronDialog;

    async function applyDbPath(dbPath: string) {
        setError(null);
        setLoading(true);
        try {
            await saveDbPath(dbPath);
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

    async function handleCreateNew() {
        if (!isElectron) {
            await applyDbPath(manualPath.trim());
            return;
        }
        const selectedPath = await window.electronDialog!.selectNewDbPath();
        if (selectedPath) {
            await applyDbPath(selectedPath);
        }
    }

    async function handleOpenExisting() {
        if (!isElectron) {
            await applyDbPath(manualPath.trim());
            return;
        }
        const selectedPath = await window.electronDialog!.selectExistingDbPath();
        if (selectedPath) {
            await applyDbPath(selectedPath);
        }
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
            <Card sx={{ width: '100%', maxWidth: 500, p: 2 }} elevation={6}>
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

                    <Box display="flex" flexDirection="column" gap={2}>
                        <Button
                            variant="contained"
                            size="large"
                            startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <Storage />}
                            onClick={handleCreateNew}
                            disabled={loading || (!isElectron && !manualPath.trim())}
                            fullWidth
                        >
                            {t('dbSetup.createNew')}
                        </Button>

                        <Divider>{t('dbSetup.or')}</Divider>

                        <Button
                            variant="outlined"
                            size="large"
                            startIcon={<FolderOpen />}
                            onClick={handleOpenExisting}
                            disabled={loading || (!isElectron && !manualPath.trim())}
                            fullWidth
                        >
                            {t('dbSetup.openExisting')}
                        </Button>

                        {!isElectron && (
                            <TextField
                                label={t('dbSetup.manualPathLabel')}
                                variant="outlined"
                                fullWidth
                                value={manualPath}
                                onChange={(e) => setManualPath(e.target.value)}
                                placeholder={t('dbSetup.manualPathPlaceholder')}
                                helperText={t('dbSetup.manualPathHelper')}
                            />
                        )}

                        {error && (
                            <Alert severity="error" variant="outlined">
                                {error}
                            </Alert>
                        )}
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default DbSetup;
