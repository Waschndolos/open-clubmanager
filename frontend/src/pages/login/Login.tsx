import React, {useState, useEffect} from 'react';
import {Alert, Box, Button, Card, CardContent, IconButton, InputAdornment, Link, TextField, Typography,} from '@mui/material';
import {Visibility, VisibilityOff} from '@mui/icons-material';
import {useAuth} from '../../context/AuthContext';
import {useNavigate, useLocation} from 'react-router-dom';
import {login} from '../../api/authentication';
import {getSetupStatus} from '../../api/setup';
import {useTranslation} from "react-i18next";
import {setAccessToken} from "../../api/api";
import loginBg from '../../assets/login_bg.jpeg';

const Login: React.FC = () => {
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

    const setupState = (location.state as { setupComplete?: boolean; setupWarning?: string } | null) ?? null;
    const setupComplete = setupState?.setupComplete;
    const setupWarning = setupState?.setupWarning;

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
                                sx={{
                                    letterSpacing: '-0.02em',
                                    mb: 1,
                                }}
                            >
                                {t('login.title')}
                            </Typography>
                            <Typography variant="body2" color="text.secondary">
                                {t('login.welcomeSubtitle')}
                            </Typography>
                        </Box>

                        <Box display="flex" flexDirection="column" gap={2}>
                            {setupComplete && (
                                <Alert severity="success" variant="outlined">
                                    {t('login.setupComplete')}
                                </Alert>
                            )}
                            {setupWarning && (
                                <Alert severity="warning" variant="outlined">
                                    {setupWarning}
                                </Alert>
                            )}

                            <TextField
                                label={t('login.email')}
                                variant="outlined"
                                fullWidth
                                value={email}
                                onChange={(e) => setEmail(e.target.value)}
                                onKeyDown={handleKeyDown}
                            />

                            <TextField
                                label={t('login.password')}
                                variant="outlined"
                                fullWidth
                                type={showPassword ? 'text' : 'password'}
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                onKeyDown={handleKeyDown}
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
                                                    {showPassword ? <VisibilityOff fontSize="small"/> : <Visibility fontSize="small"/>}
                                                </IconButton>
                                            </InputAdornment>
                                        ),
                                    },
                                }}
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
                                onClick={handleLogin}
                                disabled={!email || !password}
                                sx={{ mt: 0.5 }}
                            >
                                {t('login.loginButton')}
                            </Button>

                            <Link
                                component="button"
                                variant="body2"
                                onClick={() => navigate('/forgot-password')}
                                sx={{
                                    alignSelf: 'center',
                                    color: 'primary.main',
                                    fontWeight: 600,
                                    textDecoration: 'none',
                                    '&:hover': { textDecoration: 'underline' },
                                }}
                            >
                                {t('login.forgotPassword')}
                            </Link>
                        </Box>
                    </CardContent>
                </Card>
            </Box>
        </Box>
    );
};

export default Login;
