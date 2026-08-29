import React, { useState } from 'react';
import { Alert, Box, Button, Card, CardContent, TextField, Typography } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { forgotPassword } from '../../api/authentication';
import { useTranslation } from 'react-i18next';
import loginBg from '../../assets/login_bg.jpeg';

const ForgotPassword: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [email, setEmail] = useState('');
    const [submitted, setSubmitted] = useState(false);
    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
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
                                {t('forgotPassword.title')}
                            </Typography>
                            {!submitted && (
                                <Typography variant="body2" color="text.secondary">
                                    {t('forgotPassword.description')}
                                </Typography>
                            )}
                        </Box>

                        <Box display="flex" flexDirection="column" gap={2}>
                            {submitted ? (
                                <>
                                    <Alert severity="success" variant="outlined">
                                        {t('forgotPassword.successMessage')}
                                    </Alert>
                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={() => navigate('/login')}
                                    >
                                        {t('forgotPassword.backToLogin')}
                                    </Button>
                                </>
                            ) : (
                                <>
                                    <TextField
                                        label={t('forgotPassword.email')}
                                        variant="outlined"
                                        fullWidth
                                        type="email"
                                        value={email}
                                        onChange={(e) => setEmail(e.target.value)}
                                    />

                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={handleSubmit}
                                        disabled={!email || loading}
                                    >
                                        {t('forgotPassword.submitButton')}
                                    </Button>

                                    <Button
                                        variant="text"
                                        fullWidth
                                        onClick={() => navigate('/login')}
                                    >
                                        {t('forgotPassword.backToLogin')}
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

export default ForgotPassword;
