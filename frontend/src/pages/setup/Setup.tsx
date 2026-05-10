import React, { useState, useEffect } from 'react';
import {
    Box,
    Button,
    Card,
    CardContent,
    IconButton,
    InputAdornment,
    TextField,
    Typography,
    Alert,
    Stepper,
    Step,
    StepLabel,
    ToggleButton,
    ToggleButtonGroup,
} from '@mui/material';
import { Visibility, VisibilityOff, AdminPanelSettings } from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useThemeContext } from '../../theme/ThemeContext';
import {
    configureDatabase,
    initializeAdmin,
    getSetupStatus,
    SetupDatabaseMode,
} from '../../api/setup';

const Setup: React.FC = () => {
    const { mode } = useThemeContext();
    const { t } = useTranslation();
    const navigate = useNavigate();

    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [databaseMode, setDatabaseMode] = useState<SetupDatabaseMode>('sqlite-local');
    const [databaseUrl, setDatabaseUrl] = useState('');
    const [databaseConfigured, setDatabaseConfigured] = useState(false);

    useEffect(() => {
        getSetupStatus()
            .then(({ setupRequired, databaseConfigured: isConfigured, databaseMode: mode }) => {
                if (!setupRequired) {
                    navigate('/login', { replace: true });
                    return;
                }

                setDatabaseMode(mode);
                setDatabaseConfigured(isConfigured);
                if (isConfigured && mode === 'sqlite-local') {
                    setDatabaseUrl('file:./clubmanager.db');
                }
            })
            .catch(() => {
                // If setup status cannot be fetched, allow setup page to render
            });
    }, [navigate]);

    const passwordsMatch = password === confirmPassword;
    const passwordLongEnough = password.length >= 8;
    const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    const canSubmit = emailValid && passwordLongEnough && passwordsMatch && !submitting;
    const needsDatabaseUrl = databaseMode === 'mysql-shared';
    const databaseUrlValid = needsDatabaseUrl
        ? /^mysqls?:\/\//i.test(databaseUrl.trim())
        : true;

    const inputSx = {
        '& input': {
            transition: 'background-color 5000s ease-in-out 0s',
            backgroundColor: 'transparent',
            WebkitTextFillColor: mode === 'dark' ? 'white' : 'black',
            MozTextFillColor: mode === 'dark' ? 'white' : 'black',
            color: mode === 'dark' ? 'white' : 'black',
        },
    };

    const handleConfigureDatabase = async () => {
        setError(null);
        setSubmitting(true);
        try {
            await configureDatabase(databaseMode, needsDatabaseUrl ? databaseUrl.trim() : undefined);
            setDatabaseConfigured(true);
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('setup.error.generic'));
        } finally {
            setSubmitting(false);
        }
    };

    const handleSubmit = async () => {
        setError(null);
        setSubmitting(true);
        try {
            await initializeAdmin(email, password);
            navigate('/login', { replace: true, state: { setupComplete: true } });
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('setup.error.generic'));
        } finally {
            setSubmitting(false);
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
            <Card sx={{ width: '100%', maxWidth: 480, p: 2 }} elevation={6}>
                <CardContent>
                    <Box display="flex" flexDirection="column" alignItems="center" mb={2}>
                        <AdminPanelSettings sx={{ fontSize: 48, color: 'primary.main', mb: 1 }} />
                        <Typography variant="h5" component="h1" gutterBottom align="center">
                            {t('setup.title')}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" align="center">
                            {t('setup.description')}
                        </Typography>
                    </Box>

                    <Stepper activeStep={databaseConfigured ? 1 : 0} sx={{ mb: 3 }}>
                        <Step completed={false}>
                            <StepLabel>{t('setup.steps.database')}</StepLabel>
                        </Step>
                        <Step completed={false}>
                            <StepLabel>{t('setup.steps.createAdmin')}</StepLabel>
                        </Step>
                        <Step>
                            <StepLabel>{t('setup.steps.login')}</StepLabel>
                        </Step>
                    </Stepper>

                    <Box display="flex" flexDirection="column" gap={2}>
                        {!databaseConfigured && (
                            <>
                                <ToggleButtonGroup
                                    value={databaseMode}
                                    exclusive
                                    onChange={(_event, value: SetupDatabaseMode | null) => {
                                        if (value) {
                                            setDatabaseMode(value);
                                        }
                                    }}
                                    fullWidth
                                    color="primary"
                                >
                                    <ToggleButton value="sqlite-local">
                                        {t('setup.database.localOption')}
                                    </ToggleButton>
                                    <ToggleButton value="mysql-shared">
                                        {t('setup.database.sharedOption')}
                                    </ToggleButton>
                                </ToggleButtonGroup>

                                <Typography variant="body2" color="text.secondary">
                                    {databaseMode === 'sqlite-local'
                                        ? t('setup.database.localHint')
                                        : t('setup.database.sharedHint')}
                                </Typography>

                                {needsDatabaseUrl && (
                                    <TextField
                                        label={t('setup.database.urlLabel')}
                                        variant="outlined"
                                        fullWidth
                                        value={databaseUrl}
                                        onChange={(e) => setDatabaseUrl(e.target.value)}
                                        error={databaseUrl.length > 0 && !databaseUrlValid}
                                        helperText={
                                            databaseUrl.length > 0 && !databaseUrlValid
                                                ? t('setup.validation.invalidMysqlUrl')
                                                : t('setup.database.urlHint')
                                        }
                                        slotProps={{ input: { sx: inputSx } }}
                                    />
                                )}

                                <Button
                                    variant="contained"
                                    fullWidth
                                    onClick={handleConfigureDatabase}
                                    disabled={submitting || !databaseUrlValid}
                                >
                                    {submitting ? t('setup.database.saving') : t('setup.database.saveButton')}
                                </Button>
                            </>
                        )}

                        {databaseConfigured && (
                            <>
                                <Alert severity="success" variant="outlined">
                                    {t('setup.database.configured')}
                                </Alert>

                                <TextField
                                    label={t('setup.email')}
                                    variant="outlined"
                                    fullWidth
                                    type="email"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    error={email.length > 0 && !emailValid}
                                    helperText={email.length > 0 && !emailValid ? t('setup.validation.invalidEmail') : ''}
                                    slotProps={{ input: { sx: inputSx } }}
                                />

                                <TextField
                                    label={t('setup.password')}
                                    variant="outlined"
                                    fullWidth
                                    type={showPassword ? 'text' : 'password'}
                                    value={password}
                                    onChange={(e) => setPassword(e.target.value)}
                                    error={password.length > 0 && !passwordLongEnough}
                                    helperText={
                                        password.length > 0 && !passwordLongEnough
                                            ? t('setup.validation.passwordTooShort')
                                            : ''
                                    }
                                    slotProps={{
                                        input: {
                                            sx: inputSx,
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
                                    label={t('setup.confirmPassword')}
                                    variant="outlined"
                                    fullWidth
                                    type={showConfirmPassword ? 'text' : 'password'}
                                    value={confirmPassword}
                                    onChange={(e) => setConfirmPassword(e.target.value)}
                                    error={confirmPassword.length > 0 && !passwordsMatch}
                                    helperText={
                                        confirmPassword.length > 0 && !passwordsMatch
                                            ? t('setup.validation.passwordMismatch')
                                            : ''
                                    }
                                    slotProps={{
                                        input: {
                                            sx: inputSx,
                                            endAdornment: (
                                                <InputAdornment position="end">
                                                    <IconButton
                                                        onClick={() => setShowConfirmPassword((prev) => !prev)}
                                                        edge="end"
                                                        aria-label="toggle confirm password visibility"
                                                    >
                                                        {showConfirmPassword ? <VisibilityOff /> : <Visibility />}
                                                    </IconButton>
                                                </InputAdornment>
                                            ),
                                        },
                                    }}
                                />

                                <Button
                                    variant="contained"
                                    fullWidth
                                    onClick={handleSubmit}
                                    disabled={!canSubmit}
                                >
                                    {submitting ? t('setup.creating') : t('setup.createButton')}
                                </Button>
                            </>
                        )}

                        {error && (
                            <Alert severity="error" variant="outlined">
                                {error}
                            </Alert>
                        )}
                    </Box>
                </CardContent>
            </Card>
        </Box>
    );
};

export default Setup;
