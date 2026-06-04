import React, {useState, useEffect} from 'react';
import {Alert, Box, Button, Card, CardContent, IconButton, InputAdornment, Link, TextField, Typography,} from '@mui/material';
import {Visibility, VisibilityOff} from '@mui/icons-material';
import {useAuth} from '../../context/AuthContext';
import {useNavigate, useLocation} from 'react-router-dom';
import {login} from '../../api/authentication';
import {getSetupStatus} from '../../api/setup';
import {useTranslation} from "react-i18next";
import {useThemeContext} from "../../theme/ThemeContext";
import {setAccessToken} from "../../api/api";
import { useTheme } from '@mui/material/styles';
import loginBg from '../../assets/login_bg.jpeg';

const Login: React.FC = () => {
    const { mode } = useThemeContext();
    const theme = useTheme();
    const { t } = useTranslation();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const {setAccessToken: setAuthAccessToken} = useAuth();
    const navigate = useNavigate();
    const location = useLocation();

    useEffect(() => {

        getSetupStatus()
            .then(({ setupRequired }) => {
                if (setupRequired) {
                    navigate('/setup', { replace: true });
                }
            })
            .catch(() => {
                // If setup status cannot be fetched, allow login page to render normally
            });
    }, [navigate, setAuthAccessToken]);

    const setupComplete = (location.state as { setupComplete?: boolean } | null)?.setupComplete;

    const handleLogin = async () => {
        setError(null);
        try {
            const {accessToken} = await login(email, password);
            setAccessToken(accessToken);
            setAuthAccessToken(accessToken);
            navigate('/dashboard', {replace: true});
        } catch (e) {
            console.log('Login error', e);
            setError(t('login.errorInvalidCredentials'));
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && email && password) {
            handleLogin();
        }
    };

    const textPrimary = theme.palette.text.primary;
    const textSecondary = theme.palette.text.secondary;
    const accentStrong = theme.palette.primary.main;
    const backgroundLight = mode === 'dark' ? 'rgba(24, 30, 32, 0.72)' : 'rgba(236, 249, 246, 0.62)';
    const welcomeTextColor = mode === 'dark' ? '#F0FAFF' : '#1D3A36';
    const welcomeSubtextColor = mode === 'dark' ? 'rgba(224, 245, 255, 0.92)' : 'rgba(29, 58, 54, 0.88)';

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
                    ? 'rgba(181, 209, 203, 0.30)'
                    : 'rgba(45, 52, 54, 0.28)',
            },
            '&:hover fieldset': {
                borderColor: theme.palette.mode === 'dark'
                    ? 'rgba(181, 209, 203, 0.38)'
                    : 'rgba(45, 52, 54, 0.34)',
            },
            '&.Mui-focused fieldset': {
                borderColor: theme.palette.mode === 'dark'
                    ? 'rgba(181, 209, 203, 0.46)'
                    : 'rgba(45, 52, 54, 0.40)',
                borderWidth: 1,
            },
        },
    };

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
                        borderRight: mode === 'dark' ? '1px solid rgba(0, 255, 194, 0.24)' : '1px solid rgba(0, 200, 154, 0.18)',
                        backdropFilter: 'blur(14px)',
                    }}
                >
                    <Card
                        sx={{
                            width: '100%',
                            maxWidth: 520,
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
                                {t('login.title')}
                            </Typography>

                            <Box display="flex" flexDirection="column" gap={2} mt={2} alignItems={"center"}>
                                {setupComplete && (
                                    <Alert severity="success" variant="outlined" sx={{ width: '100%' }}>
                                        {t('login.setupComplete')}
                                    </Alert>
                                )}
                                <TextField
                                    label={t('login.email')}
                                    variant="outlined"
                                    fullWidth
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    onKeyDown={handleKeyDown}
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

                                <TextField
                                    label={t('login.password')}
                                    variant="outlined"
                                    fullWidth
                                    type={showPassword ? 'text' : 'password'}
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    onKeyDown={handleKeyDown}
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
                                                        sx={{ color: textSecondary }}
                                                    >
                                                        {showPassword ? <VisibilityOff/> : <Visibility/>}
                                                    </IconButton>
                                                </InputAdornment>
                                            ),
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
                                    onClick={handleLogin}
                                    disabled={!email || !password}
                                    sx={{
                                        mt: 0.5,
                                        py: 1.2,
                                        fontWeight: 700,
                                        bgcolor: accentStrong,
                                        color: '#0D1B18',
                                        '&:hover': { bgcolor: '#00B287' },
                                    }}
                                >
                                    {t('login.loginButton')}
                                </Button>

                                <Link
                                    component="button"
                                    variant="body2"
                                    onClick={() => navigate('/forgot-password')}
                                    sx={{ alignSelf: 'center', color: accentStrong, fontWeight: 600 }}
                                >
                                    {t('login.forgotPassword')}
                                </Link>
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
                                color: welcomeTextColor,
                                fontWeight: 800,
                                lineHeight: 1.06,
                                letterSpacing: '-0.02em',
                                mb: 2,
                                textShadow: '0 8px 30px rgba(0,0,0,0.45)',
                            }}
                        >
                            {t('login.welcomeTitle')}
                        </Typography>
                        <Typography
                            variant="h6"
                            sx={{
                                color: welcomeSubtextColor,
                                fontWeight: 400,
                                lineHeight: 1.5,
                                textShadow: '0 4px 18px rgba(0,0,0,0.35)',
                            }}
                        >
                            {t('login.welcomeSubtitle')}
                        </Typography>
                    </Box>
                </Box>
            </Box>
        </Box>
    );
};

export default Login;
