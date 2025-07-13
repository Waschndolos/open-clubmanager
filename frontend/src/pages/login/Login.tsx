import React, {useState} from 'react';
import {
    Box,
    Button,
    Card,
    CardContent,
    TextField,
    Typography,
    InputAdornment,
    IconButton,
} from '@mui/material';
import {Visibility, VisibilityOff} from '@mui/icons-material';
import {useAuth} from '../../context/AuthContext';
import {useNavigate} from 'react-router-dom';
import {login} from '../../api/authentication';

const Login: React.FC = () => {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const {setAccessToken} = useAuth();
    const navigate = useNavigate();

    const handleLogin = async () => {
        setError(null);
        try {
            const {accessToken} = await login(email, password);
            setAccessToken(accessToken);
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
                        Anmelden
                    </Typography>

                    <Box display="flex" flexDirection="column" gap={2} mt={2} alignItems={"center"}>
                        <TextField
                            label="E-Mail"
                            variant="outlined"
                            fullWidth
                            value={email}
                            onChange={(e) => setEmail(e.target.value)}
                        />

                        <TextField
                            label="Passwort"
                            variant="outlined"
                            fullWidth
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
                            Einloggen
                        </Button>
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default Login;
