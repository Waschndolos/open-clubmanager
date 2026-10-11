import React, {useState, useEffect} from 'react';
import {Alert, Box, Button, CircularProgress, IconButton, InputAdornment, Link, TextField} from '@mui/material';
import {Visibility, VisibilityOff} from '@mui/icons-material';
import {useAuth} from '../../context/AuthContext';
import {useNavigate, useLocation} from 'react-router-dom';
import {login} from '../../api/authentication';
import {getSetupStatus} from '../../api/setup';
import {useTranslation} from "react-i18next";
import {setAccessToken} from "../../api/api";
import AuthLayout from '../../components/auth/AuthLayout';

const Login: React.FC = () => {
    const { t } = useTranslation();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

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

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!email || !password || submitting) return;
        setError(null);
        setSubmitting(true);
        try {
            const {accessToken} = await login(email, password);
            setAccessToken(accessToken);
            setAuthAccessToken(accessToken);
            navigate('/dashboard', {replace: true});
        } catch (e) {
            console.log('Login error', e);
            setError(t('login.errorInvalidCredentials'));
            setSubmitting(false);
        }
    };

    return (
        <AuthLayout title={t('login.title')} subtitle={t('login.welcomeSubtitle')}>
            <Box component="form" onSubmit={handleLogin} noValidate sx={{ display: 'flex', flexDirection: 'column', gap: 2.25 }}>
                {setupComplete && <Alert severity="success">{t('login.setupComplete')}</Alert>}
                {setupWarning && <Alert severity="warning">{setupWarning}</Alert>}

                <TextField
                    label={t('login.email')}
                    type="email"
                    autoComplete="username"
                    autoFocus
                    fullWidth
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                />

                <TextField
                    label={t('login.password')}
                    fullWidth
                    autoComplete="current-password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
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
                                        {showPassword ? <VisibilityOff fontSize="small"/> : <Visibility fontSize="small"/>}
                                    </IconButton>
                                </InputAdornment>
                            ),
                        },
                    }}
                />

                <Link
                    component="button"
                    type="button"
                    variant="body2"
                    onClick={() => navigate('/forgot-password')}
                    underline="hover"
                    sx={{ alignSelf: 'flex-end', mt: -1, fontWeight: 600 }}
                >
                    {t('login.forgotPassword')}
                </Link>

                {error && <Alert severity="error">{error}</Alert>}

                <Button
                    type="submit"
                    variant="contained"
                    fullWidth
                    size="large"
                    disabled={!email || !password || submitting}
                    startIcon={submitting ? <CircularProgress size={18} color="inherit" /> : undefined}
                >
                    {t('login.loginButton')}
                </Button>
            </Box>
        </AuthLayout>
    );
};

export default Login;
