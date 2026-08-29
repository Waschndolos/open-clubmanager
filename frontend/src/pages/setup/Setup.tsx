import React, { useState, useEffect } from 'react';
import {
    Box,
    Button,
    Card,
    CardContent,
    Checkbox,
    FormControlLabel,
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
import {
    configureDatabase,
    initializeAdmin,
    getSetupStatus,
    SetupDatabaseMode,
} from '../../api/setup';
import { login } from '../../api/authentication';
import { setAccessToken } from '../../api/api';
import { seedSetupDemoData } from '../../api/setupDemoData';
import { getDatabaseSettings } from '../../api/settings';
import { validatePath } from '../../api/validation';
import loginBg from '../../assets/login_bg.jpeg';

const Setup: React.FC = () => {
    const { t } = useTranslation();
    const navigate = useNavigate();

    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [confirmPassword, setConfirmPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const [currentStep, setCurrentStep] = useState(0);
    const [databaseMode, setDatabaseMode] = useState<SetupDatabaseMode>('sqlite-local');
    const [databaseUrl, setDatabaseUrl] = useState('');
    const [databaseConfigured, setDatabaseConfigured] = useState(false);
    const [databaseValidationMessage, setDatabaseValidationMessage] = useState('');
    const [importDemoData, setImportDemoData] = useState(false);

    const isWindowsClient = typeof navigator !== 'undefined' && /windows/i.test(navigator.userAgent);
    const localPathPlaceholder = isWindowsClient
        ? 'C:\\Users\\Name\\clubmanager.db'
        : '/home/user/clubmanager.db';

    useEffect(() => {
        getSetupStatus()
            .then(({ setupRequired, databaseConfigured: isConfigured, databaseMode: mode }) => {
                if (!setupRequired) {
                    navigate('/login', { replace: true });
                    return;
                }

                setDatabaseMode(mode);
                setDatabaseConfigured(isConfigured);
                return getDatabaseSettings()
                    .then(({ databaseUrl: url }) => {
                        setDatabaseUrl(url);
                    })
                    .catch(() => {
                        if (mode === 'sqlite-local') {
                            setDatabaseUrl('file:./clubmanager.db');
                        }
                    });
            })
            .catch(() => {
                // If setup status cannot be fetched, allow setup page to render
            });
    }, [navigate]);

    const passwordsMatch = password === confirmPassword;
    const passwordLongEnough = password.length >= 8;
    const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
    const canSubmit = emailValid && passwordLongEnough && passwordsMatch && !submitting;
    const isMysqlMode = databaseMode === 'mysql-shared';
    const currentDatabaseValue = databaseUrl.trim();
    const databaseUrlValid = isMysqlMode
        ? /^mysqls?:\/\//i.test(currentDatabaseValue)
        : currentDatabaseValue.length > 0;

    const handleConfigureDatabase = async () => {
        setError(null);
        setSubmitting(true);
        try {
            const configuredValue = isMysqlMode
                ? currentDatabaseValue
                : currentDatabaseValue.startsWith('file:')
                    ? currentDatabaseValue
                    : `file:${currentDatabaseValue}`;

            await configureDatabase(databaseMode, configuredValue);
            setDatabaseConfigured(true);
            setCurrentStep(1);
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('setup.error.generic'));
        } finally {
            setSubmitting(false);
        }
    };

    const validateDatabaseInput = async () => {
        if (!currentDatabaseValue) {
            setDatabaseValidationMessage('');
            return;
        }

        try {
            if (isMysqlMode) {
                setDatabaseValidationMessage(
                    /^mysqls?:\/\//i.test(currentDatabaseValue)
                        ? ''
                        : t('settings.validatíon.error.invalidmysqlurl')
                );
                return;
            }

            const localPath = currentDatabaseValue.startsWith('file:')
                ? currentDatabaseValue.replace('file:', '').replace(/^\/\//, '')
                : currentDatabaseValue;
            const response = await validatePath(localPath);
            setDatabaseValidationMessage(
                response.valid
                    ? ''
                    : t(`settings.validatíon.${response.i18nToken}`)
            );
        } catch {
            setDatabaseValidationMessage('');
        }
    };

    const handleSubmit = async () => {
        setError(null);
        setSubmitting(true);
        try {
            await initializeAdmin(email, password);

            let setupWarning: string | undefined;
            if (importDemoData) {
                try {
                    const { accessToken } = await login(email, password);
                    setAccessToken(accessToken);
                    await seedSetupDemoData();
                } catch {
                    setupWarning = t('setup.demoData.warningContinue');
                } finally {
                    setAccessToken(null);
                }
            }

            navigate('/login', { replace: true, state: { setupComplete: true, setupWarning } });
        } catch (e: unknown) {
            setError(e instanceof Error ? e.message : t('setup.error.generic'));
        } finally {
            setSubmitting(false);
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
                py: 4,
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
                    maxWidth: 520,
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
                        <Box display="flex" flexDirection="column" alignItems="center" mb={3}>
                            <AdminPanelSettings
                                sx={{
                                    fontSize: 48,
                                    color: 'primary.main',
                                    mb: 1,
                                }}
                            />
                            <Typography
                                variant="h4"
                                component="h1"
                                fontWeight={800}
                                sx={{ letterSpacing: '-0.02em', mb: 0.5 }}
                            >
                                {t('setup.title')}
                            </Typography>
                            <Typography variant="body2" color="text.secondary" align="center">
                                {t('setup.description')}
                            </Typography>
                        </Box>

                        <Stepper activeStep={currentStep} sx={{ mb: 3 }}>
                            <Step completed={false}>
                                <StepLabel>{t('setup.steps.database')}</StepLabel>
                            </Step>
                            <Step completed={false}>
                                <StepLabel>{t('setup.steps.createAdmin')}</StepLabel>
                            </Step>
                        </Stepper>

                        <Box display="flex" flexDirection="column" gap={2}>
                            {currentStep === 0 && (
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
                                        {isMysqlMode
                                            ? t('setup.database.sharedHint')
                                            : t('setup.database.localHint')}
                                    </Typography>

                                    <TextField
                                        label={isMysqlMode ? t('setup.database.urlLabel') : t('setup.database.localPathLabel')}
                                        variant="outlined"
                                        fullWidth
                                        value={databaseUrl}
                                        onChange={(e) => {
                                            setDatabaseUrl(e.target.value);
                                            setDatabaseValidationMessage('');
                                        }}
                                        onBlur={validateDatabaseInput}
                                        error={databaseValidationMessage.length > 0 || (databaseUrl.length > 0 && !databaseUrlValid)}
                                        helperText={
                                            databaseValidationMessage || (isMysqlMode
                                                ? t('setup.database.mysqlPlaceholderHint')
                                                : t('setup.database.localPathHint'))
                                        }
                                        placeholder={isMysqlMode
                                            ? 'mysql://user:password@localhost:3306/clubmanager'
                                            : localPathPlaceholder}
                                    />

                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={handleConfigureDatabase}
                                        disabled={submitting || !databaseUrlValid}
                                    >
                                        {submitting
                                            ? t('setup.database.saving')
                                            : databaseConfigured
                                                ? t('setup.database.continueButton')
                                                : t('setup.database.saveButton')}
                                    </Button>
                                </>
                            )}

                            {currentStep === 1 && (
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
                                        error={email.length > 0 && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)}
                                        helperText={email.length > 0 && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) ? t('setup.validation.invalidEmail') : ''}
                                    />

                                    <TextField
                                        label={t('setup.password')}
                                        variant="outlined"
                                        fullWidth
                                        type={showPassword ? 'text' : 'password'}
                                        value={password}
                                        onChange={(e) => setPassword(e.target.value)}
                                        error={password.length > 0 && password.length < 8}
                                        helperText={
                                            password.length > 0 && password.length < 8
                                                ? t('setup.validation.passwordTooShort')
                                                : ''
                                        }
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
                                                            {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
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
                                        error={confirmPassword.length > 0 && password !== confirmPassword}
                                        helperText={
                                            confirmPassword.length > 0 && password !== confirmPassword
                                                ? t('setup.validation.passwordMismatch')
                                                : ''
                                        }
                                        slotProps={{
                                            input: {
                                                endAdornment: (
                                                    <InputAdornment position="end">
                                                        <IconButton
                                                            onClick={() => setShowConfirmPassword((prev) => !prev)}
                                                            edge="end"
                                                            aria-label="toggle confirm password visibility"
                                                            size="small"
                                                        >
                                                            {showConfirmPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
                                                        </IconButton>
                                                    </InputAdornment>
                                                ),
                                            },
                                        }}
                                    />

                                    <FormControlLabel
                                        control={(
                                            <Checkbox
                                                checked={importDemoData}
                                                onChange={(event) => setImportDemoData(event.target.checked)}
                                                disabled={submitting}
                                            />
                                        )}
                                        label={t('setup.demoData.label')}
                                    />

                                    <Typography variant="body2" color="text.secondary">
                                        {t('setup.demoData.hint')}
                                    </Typography>

                                    <Button
                                        variant="contained"
                                        fullWidth
                                        size="large"
                                        onClick={handleSubmit}
                                        disabled={!canSubmit}
                                    >
                                        {submitting
                                            ? (importDemoData ? t('setup.demoData.seeding') : t('setup.creating'))
                                            : t('setup.createButton')}
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
        </Box>
    );
};

export default Setup;
