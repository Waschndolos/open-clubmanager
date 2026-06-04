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
import { useTheme } from '@mui/material/styles';
import loginBg from '../../assets/login_bg.jpeg';

const ResetPassword: React.FC = () => {
    const { mode } = useThemeContext();
    const theme = useTheme();
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

    const textPrimary = theme.palette.text.primary;
    const textSecondary = theme.palette.text.secondary;
    const backgroundLight = mode === 'dark' ? '#162122' : '#FFFFFF';

    const textFieldSx = {
        '& .MuiInputLabel-root': {
            color: textSecondary,
        },
        '& .MuiInputLabel-root.Mui-focused': {
            color: textSecondary,
        },
        '& .MuiOutlinedInput-root': {
            color: textPrimary,
            '& fieldset': {
                borderColor: theme.palette.mode === 'dark'
                    ? '#203436'
                    : '#203436',
            },
            '&:hover fieldset': {
                borderColor: '#00FFC2',
            },
            '&.Mui-focused fieldset': {
                borderColor: '#00FFC2',
                borderWidth: 2,
            },
        },
    };

    if (!token) {
        return (
            <Box
                sx={{
                    position: 'relative',
                    minHeight: '100vh',
                    minWidth: '100vw',
                    overflow: 'hidden',
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
                        filter: 'saturate(1.2) contrast(1.05) brightness(0.78) hue-rotate(-14deg)',
                        transform: 'scale(1.04)',
                    }}
                />

                <Box
                    sx={{
                        position: 'absolute',
                        inset: 0,
                        background: 'linear-gradient(100deg, rgba(16,20,21,0.28) 0%, rgba(24,30,32,0.44) 50%, rgba(22,28,30,0.78) 100%)',
                    }}
                />

                <Box
                    sx={{
                        position: 'relative',
                        minHeight: '100vh',
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                    }}
                >
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
                    filter: 'saturate(1.2) contrast(1.05) brightness(0.78) hue-rotate(-14deg)',
                    transform: 'scale(1.04)',
                }}
            />

            <Box
                sx={{
                    position: 'absolute',
                    inset: 0,
                    background: 'linear-gradient(100deg, rgba(16,20,21,0.28) 0%, rgba(24,30,32,0.44) 50%, rgba(22,28,30,0.78) 100%)',
                }}
            />

            <Box
                sx={{
                    position: 'relative',
                    minHeight: '100vh',
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'stretch',
                }}
            >
                <Box
                    sx={{
                        height: '100vh',
                        width: { xs: '100%', md: '40vw' },
                        minWidth: { md: 460 },
                        maxWidth: { md: 760 },
                        px: { xs: 2, md: 5 },
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        bgcolor: backgroundLight,
                        borderRight: '1px solid #203436',
                        backdropFilter: 'blur(14px)',
                    }}
                >
                    <Card
                        sx={{
                            width: '100%',
                            maxWidth: 480,
                            p: { xs: 1.5, md: 2 },
                            bgcolor: 'transparent',
                            border: 'none',
                            boxShadow: 'none',
                        }}
                        elevation={0}
                    >
                        <CardContent>
                            <Typography
                                variant="h5"
                                component="h1"
                                gutterBottom
                                align="center"
                                fontWeight={700}
                                sx={{ color: textPrimary }}
                            >
                                {t('resetPassword.title')}
                            </Typography>

                            <Box display="flex" flexDirection="column" gap={2} mt={2} alignItems={"center"}>
                                {success ? (
                                    <>
                                        <Alert severity="success" variant="outlined" sx={{ width: '100%' }}>
                                            {t('resetPassword.successMessage')}
                                        </Alert>
                                        <Button
                                            variant="contained"
                                            fullWidth
                                            onClick={() => navigate('/login')}
                                            sx={{
                                                py: 1.2,
                                                fontWeight: 700,
                                            }}
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
                                            sx={textFieldSx}
                                            slotProps={{
                                                input: {
                                                    sx: {
                                                        '& input': {
                                                            transition: 'background-color 5000s ease-in-out 0s',
                                                            backgroundColor: 'transparent',
                                                            WebkitTextFillColor: textPrimary,
                                                            MozTextFillColor: textPrimary,
                                                            color: textPrimary,
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
                                            sx={textFieldSx}
                                            slotProps={{
                                                input: {
                                                    sx: {
                                                        '& input': {
                                                            transition: 'background-color 5000s ease-in-out 0s',
                                                            backgroundColor: 'transparent',
                                                            WebkitTextFillColor: textPrimary,
                                                            MozTextFillColor: textPrimary,
                                                            color: textPrimary,
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
                                            sx={{
                                                mt: 0.5,
                                                py: 1.2,
                                                fontWeight: 700,
                                            }}
                                        >
                                            {t('resetPassword.submitButton')}
                                        </Button>

                                        <Button
                                            variant="text"
                                            fullWidth
                                            onClick={() => navigate('/login')}
                                            sx={{ color: 'primary.main' }}
                                        >
                                            {t('resetPassword.backToLogin')}
                                        </Button>
                                    </>
                                )}
                            </Box>
                        </CardContent>
                    </Card>
                </Box>

                <Box
                    sx={{
                        display: { xs: 'none', md: 'flex' },
                        flex: 1,
                        px: { md: 7, lg: 10 },
                        py: { md: 8, lg: 10 },
                        alignItems: 'flex-end',
                    }}
                >
                    <Box sx={{ maxWidth: 640 }}>
                        <Typography
                            variant="h2"
                            sx={{
                                color: mode === 'dark' ? '#E8F4F1' : '#203436',
                                fontWeight: 800,
                                lineHeight: 1.06,
                                letterSpacing: '-0.02em',
                                mb: 2,
                                textShadow: '0 8px 30px rgba(0,0,0,0.45)',
                            }}
                        >
                            {t('resetPassword.welcomeTitle') || 'Neues Passwort setzen'}
                        </Typography>
                        <Typography
                            variant="h6"
                            sx={{
                                color: mode === 'dark' ? '#A8C5C1' : '#6B8A87',
                                fontWeight: 400,
                                lineHeight: 1.5,
                                textShadow: '0 4px 18px rgba(0,0,0,0.35)',
                            }}
                        >
                            {t('resetPassword.welcomeSubtitle') || 'Erstellen Sie ein neues Passwort für Ihren Account.'}
                        </Typography>
                    </Box>
                </Box>
            </Box>
        </Box>
    );
};

export default ResetPassword;
