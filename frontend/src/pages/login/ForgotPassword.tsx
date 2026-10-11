import React, { useState } from 'react';
import { Alert, Box, Button, TextField } from '@mui/material';
import ArrowBackRoundedIcon from '@mui/icons-material/ArrowBackRounded';
import { useNavigate } from 'react-router-dom';
import { forgotPassword } from '../../api/authentication';
import { useTranslation } from 'react-i18next';
import AuthLayout from '../../components/auth/AuthLayout';

const ForgotPassword: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [email, setEmail] = useState('');
    const [submitted, setSubmitted] = useState(false);
    const [loading, setLoading] = useState(false);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!email || loading) return;
        setLoading(true);
        try {
            await forgotPassword(email);
        } catch {
            // Always show success to avoid leaking account existence
        } finally {
            setLoading(false);
            setSubmitted(true);
        }
    };

    const backToLogin = (
        <Button
            variant={submitted ? 'contained' : 'text'}
            fullWidth
            size={submitted ? 'large' : 'medium'}
            startIcon={submitted ? undefined : <ArrowBackRoundedIcon />}
            onClick={() => navigate('/login')}
        >
            {t('forgotPassword.backToLogin')}
        </Button>
    );

    return (
        <AuthLayout
            title={t('forgotPassword.title')}
            subtitle={submitted ? undefined : t('forgotPassword.description')}
        >
            {submitted ? (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                    <Alert severity="success">{t('forgotPassword.successMessage')}</Alert>
                    {backToLogin}
                </Box>
            ) : (
                <Box component="form" onSubmit={handleSubmit} noValidate sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                    <TextField
                        label={t('forgotPassword.email')}
                        fullWidth
                        autoFocus
                        type="email"
                        autoComplete="username"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                    />
                    <Button type="submit" variant="contained" fullWidth size="large" disabled={!email || loading}>
                        {t('forgotPassword.submitButton')}
                    </Button>
                    {backToLogin}
                </Box>
            )}
        </AuthLayout>
    );
};

export default ForgotPassword;
