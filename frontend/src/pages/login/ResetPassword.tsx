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
import loginBg from '../../assets/login_bg.jpeg';

const ResetPassword: React.FC = () => {
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
                    position: 'relative',
                    minHeight: '100vh',
                    minWidth: '100vw',
                    overflow: 'hidden',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                }}
            >
                <Box
                    sx={{
                        position: 'absolute',
                        inset: 0,
                        backgroundImage: `url(${loginBg})`,
                        backgroundSize: 'cover',
                        backgroundPosition: 'center',
                        backgroundRepeat: 'no-repeat',
                        filter: 'saturate(1.1) contrast(1.05) brightness(0.65)',
                        transform: 'scale(1.02)',
                    }}
                />
                <Box
                    sx={{
                        position: 'absolute',
                        inset: 0,
                        background: 'linear-gradient(135deg, rgba(15,21,22,0.75) 0%, rgba(22,33,34,0.6) 50%, rgba(15,21,22,0.85) 100%)',
                    }}
                />
                <Box sx={{ position: 'relative' }}>
                    <Alert severity="error">{t('resetPassword.missingToken')}</Alert>
                </Box>
            </Box>
        );
    }

    return (
        <Box
            sx={{
                position: 'relative',
                minHeight: '100vh',
                minWidth: '100vw',
                overflow: 'hidden',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
            }}
        >
            <Box
                sx={{
                    position: 'absolute',
                    inset: 0,
                    backgroundImage: `url(${loginBg})`,
                    backgroundSize: 'cover',
                    backgroundPosition: 'center',
                    backgroundRepeat: 'no-repeat',
                    filter: 'saturate(1.1) contrast(1.05) brightness(0.65)',
                    transform: 'scale(1.02)',
                }}
            />

            <Box
                sx={{
                    position: 'absolute',
                    inset: 0,
                    background: 'linear-gradient(135deg, rgba(15,21,22,0.75) 0%, rgba(22,33,34,0.6) 50%, rgba(15,21,22,0.85) 100%)',
                }}
            />

            <Box
                sx={{
                    position: 'relative',
                    width: '100%',
                    maxWidth: 440,
                    mx: { xs: 2, sm: 3 },
                    my: 4,
                }}
            >
                <Card
                    sx={{
                        width: '100%',
                        p: { xs: 3, sm: 4 },
                        bgcolor: (theme) => theme.palette.mode === 'dark'
                            ? 'rgba(22, 33, 34, 0.92)'
                            : 'rgba(255, 255, 255, 0.96)',
                        border: '1px solid rgba(255, 255, 255, 0.08)',
                        boxShadow: '0 24px 60px rgba(0, 0, 0, 0.35)',
                        backdropFilter: 'blur(16px)',
                        borderRadius: 4,
                    }}
                    elevation={0}
                >
                    <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
                        <Box textAlign="center" mb={3}>
                            <Typography
                                variant="h4"
                                component="h1"
                                fontWeight={800}
                                sx={{ letterSpacing: '-0.02em', mb: 1 }}
                            >
                                {t('resetPassword.title')}
                            </Typography>
                        </Box>

                        <Box display="flex" flexDirection="column" gap={2}>
                            {success ? (
                                <>
                                    <Alert severity="success" variant="outlined">
                                        {t('resetPassword.successMessage')}
                                    </Alert>
                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={() => navigate('/login')}
                                    >
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
                                                endAdornment: (
                                                    <InputAdornment position="end">
                                                        <IconButton
                                                            onClick={() => setShowPassword((prev) => !prev)}
                                                            edge="end"
                                                            aria-label="toggle password visibility"
                                                            size="small"
                                                        >
                                                            {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
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
                                    />

                                    {error && (
                                        <Alert severity="error" variant="outlined">
                                            {error}
                                        </Alert>
                                    )}

                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={handleSubmit}
                                        disabled={!newPassword || !confirmPassword || loading}
                                    >
                                        {t('resetPassword.submitButton')}
                                    </Button>

                                    <Button
                                        variant="text"
                                        fullWidth
                                        onClick={() => navigate('/login')}
                                    >
                                        {t('resetPassword.backToLogin')}
                                    </Button>
                                </>
                            )}
                        </Box>
                    </CardContent>
                </Card>
            </Box>
        </Box>
    );
};

export default ResetPassword;
