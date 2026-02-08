import React, {useState} from 'react';
import {Box, Button, Card, CardContent, IconButton, InputAdornment, TextField, Typography,} from '@mui/material';
import {Visibility, VisibilityOff} from '@mui/icons-material';
import {useAuth} from '../../context/AuthContext';
import {useNavigate} from 'react-router-dom';
import {login} from '../../api/authentication';
import {useTranslation} from "react-i18next";
import {useThemeContext} from "../../theme/ThemeContext";
import {setAccessToken} from "../../api/api";

const Login: React.FC = () => {
    const { mode } = useThemeContext();
    const { t } = useTranslation();
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const {setAccessToken: setAuthAccessToken} = useAuth();
    const navigate = useNavigate();

    const handleLogin = async () => {
        setError(null);
        try {
            const {accessToken} = await login(email, password);
            setAccessToken(accessToken);
            setAuthAccessToken(accessToken);
            navigate('/dashboard', {replace: true});
        } catch (e) {
            setError('Login fehlgeschlagen. Bitte überprüfe deine Eingaben.');
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
            <Card sx={{width: '100%', maxWidth: 400, p: 2}} elevation={6}>
                <CardContent>
                    <Typography variant="h5" component="h1" gutterBottom align="center">
                        {t('login.title')}
                    </Typography>

                    <Box display="flex" flexDirection="column" gap={2} mt={2} alignItems={"center"}>
                        <TextField
                            label={t('login.email')}
                            variant="outlined"
                            fullWidth
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                            slotProps={{
                                input: {
                                    sx: {
                                        '& input': {
                                            transition: 'background-color 5000s ease-in-out 0s',
                                            backgroundColor: 'transparent',
                                            WebkitTextFillColor: mode === 'dark' ? 'white': 'black',
                                            MozTextFillColor: mode === 'dark' ? 'white': 'black',
                                            color: mode === 'dark' ? 'white': 'black',
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

                            slotProps={{
                                input: {
                                    sx: {
                                        '& input': {
                                            transition: 'background-color 5000s ease-in-out 0s',
                                            backgroundColor: 'transparent',
                                            WebkitTextFillColor: mode === 'dark' ? 'white': 'black',
                                            MozTextFillColor: mode === 'dark' ? 'white': 'black',
                                            color: mode === 'dark' ? 'white': 'black',
                                        },
                                    },
                                    endAdornment: (
                                        <InputAdornment position="end">
                                            <IconButton
                                                onClick={() => setShowPassword((prev) => !prev)}
                                                edge="end"
                                                aria-label="toggle password visibility"
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
                        >
                            {t('login.loginButton')}
                        </Button>
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default Login;
