import React, { useState } from 'react';
import {
    Alert,
    Box,
    Button,
    Card,
    CardContent,
    IconButton,
    InputAdornment,
    TextField,
    Typography,
} from '@mui/material';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { resetPassword } from '../../api/authentication';
import { useTranslation } from 'react-i18next';
import { useThemeContext } from '../../theme/ThemeContext';

const ResetPassword: React.FC = () => {
    const { mode } = useThemeContext();
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [searchParams] = useSearchParams();
    const token = searchParams.get('token') ?? '';

    const [newPassword, setNewPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
        setError(null);

        if (newPassword !== confirmPassword) {
            setError(t('resetPassword.passwordMismatch'));
            return;
        }

        if (newPassword.length < 8) {
            setError(t('resetPassword.passwordTooShort'));
            return;
        }

        setLoading(true);
        try {
            await resetPassword(token, newPassword);
            setSuccess(true);
        } catch (e) {
            const message = e instanceof Error ? e.message : '';
            if (message.includes('expired')) {
                setError(t('resetPassword.tokenExpired'));
            } else if (message.includes('invalid') || message.includes('Invalid')) {
                setError(t('resetPassword.tokenInvalid'));
            } else {
                setError(t('resetPassword.genericError'));
            }
        } finally {
            setLoading(false);
        }
    };

    if (!token) {
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
                <Card sx={{ width: '100%', maxWidth: 400, p: 2 }} elevation={6}>
                    <CardContent>
                        <Alert severity="error">{t('resetPassword.missingToken')}</Alert>
                        <Box mt={2}>
                            <Button variant="outlined" fullWidth onClick={() => navigate('/login')}>
                                {t('resetPassword.backToLogin')}
                            </Button>
                        </Box>
                    </CardContent>
                </Card>
            </Box>
        );
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
            <Card sx={{ width: '100%', maxWidth: 400, p: 2 }} elevation={6}>
                <CardContent>
                    <Typography variant="h5" component="h1" gutterBottom align="center">
                        {t('resetPassword.title')}
                    </Typography>

                    <Box display="flex" flexDirection="column" gap={2} mt={2}>
                        {success ? (
                            <>
                                <Alert severity="success" variant="outlined">
                                    {t('resetPassword.successMessage')}
                                </Alert>
                                <Button variant="contained" fullWidth onClick={() => navigate('/login')}>
                                    {t('resetPassword.backToLogin')}
                                </Button>
                            </>
                        ) : (
                            <>
                                <TextField
                                    label={t('resetPassword.newPassword')}
                                    variant="outlined"
                                    fullWidth
                                    type={showPassword ? 'text' : 'password'}
                                    value={newPassword}
                                    onChange={(e) => setNewPassword(e.target.value)}
                                    slotProps={{
                                        input: {
                                            sx: {
                                                '& input': {
                                                    transition: 'background-color 5000s ease-in-out 0s',
                                                    backgroundColor: 'transparent',
                                                    WebkitTextFillColor: mode === 'dark' ? 'white' : 'black',
                                                    MozTextFillColor: mode === 'dark' ? 'white' : 'black',
                                                    color: mode === 'dark' ? 'white' : 'black',
                                                },
                                            },
                                            endAdornment: (
                                                <InputAdornment position="end">
                                                    <IconButton
                                                        onClick={() => setShowPassword((prev) => !prev)}
                                                        edge="end"
                                                        aria-label="toggle password visibility"
                                                    >
                                                        {showPassword ? <VisibilityOff /> : <Visibility />}
                                                    </IconButton>
                                                </InputAdornment>
                                            ),
                                        },
                                    }}
                                />

                                <TextField
                                    label={t('resetPassword.confirmPassword')}
                                    variant="outlined"
                                    fullWidth
                                    type={showPassword ? 'text' : 'password'}
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    slotProps={{
                                        input: {
                                            sx: {
                                                '& input': {
                                                    transition: 'background-color 5000s ease-in-out 0s',
                                                    backgroundColor: 'transparent',
                                                    WebkitTextFillColor: mode === 'dark' ? 'white' : 'black',
                                                    MozTextFillColor: mode === 'dark' ? 'white' : 'black',
                                                    color: mode === 'dark' ? 'white' : 'black',
                                                },
                                            },
                                        },
                                    }}
                                />

                                {error && (
                                    <Typography color="error" variant="body2">
                                        {error}
                                    </Typography>
                                )}

                                <Button
                                    variant="contained"
                                    fullWidth
                                    onClick={handleSubmit}
                                    disabled={!newPassword || !confirmPassword || loading}
                                >
                                    {t('resetPassword.submitButton')}
                                </Button>

                                <Button variant="text" fullWidth onClick={() => navigate('/login')}>
                                    {t('resetPassword.backToLogin')}
                                </Button>
                            </>
                        )}
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default ResetPassword;
