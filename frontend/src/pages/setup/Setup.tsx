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
import { useThemeContext } from '../../theme/ThemeContext';
import { useTheme } from '@mui/material/styles';
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
    const { mode } = useThemeContext();
    const theme = useTheme();
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

    const inputSx = {
        '& input': {
            transition: 'background-color 5000s ease-in-out 0s',
            backgroundColor: 'transparent',
            WebkitTextFillColor: textPrimary,
            MozTextFillColor: textPrimary,
            color: textPrimary,
        },
    };

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
                        overflowY: 'auto',
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
                            <Box display="flex" flexDirection="column" alignItems="center" mb={2}>
                                <AdminPanelSettings
                                    sx={{
                                        fontSize: 48,
                                        color: 'primary.main',
                                        mb: 1
                                    }}
                                />
                                <Typography
                                    variant="h5"
                                    component="h1"
                                    gutterBottom
                                    align="center"
                                    fontWeight={700}
                                    sx={{ color: textPrimary }}
                                >
                                    {t('setup.title')}
                                </Typography>
                                <Typography
                                    variant="body2"
                                    color="text.secondary"
                                    align="center"
                                >
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
                                            sx={textFieldSx}
                                            slotProps={{ input: { sx: inputSx } }}
                                        />

                                        <Button
                                            variant="contained"
                                            fullWidth
                                            onClick={handleConfigureDatabase}
                                            disabled={submitting || !databaseUrlValid}
                                            sx={{
                                                py: 1.2,
                                                fontWeight: 700,
                                            }}
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
                                            sx={textFieldSx}
                                            slotProps={{ input: { sx: inputSx } }}
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
                                            sx={textFieldSx}
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
                                            error={confirmPassword.length > 0 && password !== confirmPassword}
                                            helperText={
                                                confirmPassword.length > 0 && password !== confirmPassword
                                                    ? t('setup.validation.passwordMismatch')
                                                    : ''
                                            }
                                            sx={textFieldSx}
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
                                            onClick={handleSubmit}
                                            disabled={!canSubmit}
                                            sx={{
                                                py: 1.2,
                                                fontWeight: 700,
                                            }}
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
                            {t('setup.welcomeTitle') || 'Setup'}
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
                            {t('setup.welcomeSubtitle') || 'Konfigurieren Sie Ihren Open ClubManager und erstellen Sie einen Admin-Account.'}
                        </Typography>
                    </Box>
                </Box>
            </Box>
        </Box>
    );
};

export default Setup;

