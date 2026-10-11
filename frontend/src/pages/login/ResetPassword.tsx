import React, { useState } from 'react';
import {
    Alert,
    Box,
    Button,
    IconButton,
    InputAdornment,
    TextField,
} from '@mui/material';
import { Visibility, VisibilityOff } from '@mui/icons-material';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { resetPassword } from '../../api/authentication';
import { useTranslation } from 'react-i18next';
import AuthLayout from '../../components/auth/AuthLayout';

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

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!newPassword || !confirmPassword || loading) return;
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
            <AuthLayout title={t('resetPassword.title')}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                    <Alert severity="error">{t('resetPassword.missingToken')}</Alert>
                    <Button variant="contained" size="large" fullWidth onClick={() => navigate('/forgot-password')}>
                        {t('login.forgotPassword')}
                    </Button>
                </Box>
            </AuthLayout>
        );
    }

    return (
        <AuthLayout title={t('resetPassword.title')}>
            {success ? (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                    <Alert severity="success">{t('resetPassword.successMessage')}</Alert>
                    <Button variant="contained" fullWidth size="large" onClick={() => navigate('/login')}>
                        {t('resetPassword.backToLogin')}
                    </Button>
                </Box>
            ) : (
                <Box component="form" onSubmit={handleSubmit} noValidate sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                    <TextField
                        label={t('resetPassword.newPassword')}
                        fullWidth
                        autoFocus
                        autoComplete="new-password"
                        type={showPassword ? 'text' : 'password'}
                        value={newPassword}
                        onChange={(e) => setNewPassword(e.target.value)}
                        helperText={t('setup.validation.passwordTooShort')}
                        slotProps={{
                            input: {
                                endAdornment: (
                                    <InputAdornment position="end">
                                        <IconButton
                                            onClick={() => setShowPassword((prev) => !prev)}
                                            edge="end"
                                            aria-label={t(showPassword ? 'login.hidePassword' : 'login.showPassword')}
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
                        fullWidth
                        autoComplete="new-password"
                        type={showPassword ? 'text' : 'password'}
                        value={confirmPassword}
                        onChange={(e) => setConfirmPassword(e.target.value)}
                    />

                    {error && <Alert severity="error">{error}</Alert>}

                    <Button
                        type="submit"
                        variant="contained"
                        fullWidth
                        size="large"
                        disabled={!newPassword || !confirmPassword || loading}
                    >
                        {t('resetPassword.submitButton')}
                    </Button>

                    <Button variant="text" fullWidth onClick={() => navigate('/login')}>
                        {t('resetPassword.backToLogin')}
                    </Button>
                </Box>
            )}
        </AuthLayout>
    );
};

export default ResetPassword;
