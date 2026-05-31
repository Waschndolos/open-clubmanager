import React, { useState } from 'react';
import { Alert, Box, Button, Card, CardContent, TextField, Typography } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { forgotPassword } from '../../api/authentication';
import { useTranslation } from 'react-i18next';
import { useThemeContext } from '../../theme/ThemeContext';

const ForgotPassword: React.FC = () => {
    const { mode } = useThemeContext();
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
                        {t('forgotPassword.title')}
                    </Typography>

                    <Box display="flex" flexDirection="column" gap={2} mt={2}>
                        {submitted ? (
                            <>
                                <Alert severity="success" variant="outlined">
                                    {t('forgotPassword.successMessage')}
                                </Alert>
                                <Button variant="outlined" fullWidth onClick={() => navigate('/login')}>
                                    {t('forgotPassword.backToLogin')}
                                </Button>
                            </>
                        ) : (
                            <>
                                <Typography variant="body2" color="text.secondary">
                                    {t('forgotPassword.description')}
                                </Typography>

                                <TextField
                                    label={t('forgotPassword.email')}
                                    variant="outlined"
                                    fullWidth
                                    type="email"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
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

                                <Button
                                    variant="contained"
                                    fullWidth
                                    onClick={handleSubmit}
                                    disabled={!email || loading}
                                >
                                    {t('forgotPassword.submitButton')}
                                </Button>

                                <Button variant="text" fullWidth onClick={() => navigate('/login')}>
                                    {t('forgotPassword.backToLogin')}
                                </Button>
                            </>
                        )}
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default ForgotPassword;
