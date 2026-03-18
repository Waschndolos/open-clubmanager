import React, { useState } from 'react';
import {
    Box,
    Button,
    Card,
    CardContent,
    Typography,
    Alert,
    CircularProgress,
} from '@mui/material';
import { FolderOpen, CheckCircle } from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../context/AuthContext';

const FolderSetup: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { setAccessToken } = useAuth();
    const [selectedFolder, setSelectedFolder] = useState<string | null>(null);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleSelectFolder = async () => {
        setError(null);
        setLoading(true);
        try {
            const folderPath = await window.api!.club.selectFolder();
            if (!folderPath) {
                setLoading(false);
                return;
            }
            setSelectedFolder(folderPath);
            await window.api!.club.initFolder();
            setAccessToken('folder-mode');
            navigate('/dashboard', { replace: true });
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('folderSetup.error.generic'));
        } finally {
            setLoading(false);
        }
    };

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
                        <FolderOpen sx={{ fontSize: 56, color: 'primary.main', mb: 1 }} />
                        <Typography variant="h5" component="h1" gutterBottom align="center">
                            {t('folderSetup.title')}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" align="center">
                            {t('folderSetup.description')}
                        </Typography>
                    </Box>

                    {selectedFolder && (
                        <Alert severity="success" icon={<CheckCircle />} sx={{ mb: 2 }}>
                            {selectedFolder}
                        </Alert>
                    )}

                    {error && (
                        <Alert severity="error" variant="outlined" sx={{ mb: 2 }}>
                            {error}
                        </Alert>
                    )}

                    <Button
                        variant="contained"
                        fullWidth
                        startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <FolderOpen />}
                        onClick={handleSelectFolder}
                        disabled={loading}
                        size="large"
                    >
                        {loading ? t('folderSetup.selecting') : t('folderSetup.selectButton')}
                    </Button>

                    <Typography variant="caption" color="text.secondary" display="block" mt={2} align="center">
                        {t('folderSetup.hint')}
                    </Typography>
                </CardContent>
            </Card>
        </Box>
    );
};

export default FolderSetup;
