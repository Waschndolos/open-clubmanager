import React, { useEffect, useState } from 'react';
import { Alert, Box, Button, Checkbox, CircularProgress, FormControlLabel, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { FolderInfo, getLocalData, getSafetyCode, getStatus, joinClub } from '../../api/collaboration';
import FolderPicker, { folderUsable, joinFolderClub } from '../../components/collaboration/FolderPicker';
import { SafetyCodeDisplay, useErrorText, useRoleName } from '../../components/collaboration/common';
import { useCollaboration } from '../../context/CollaborationContext';
import WizardFrame from './WizardFrame';

const CODE_CHARS = /^[A-Z2-7]*$/;

export function normalizeInviteCode(code: string): string {
    return code.toUpperCase().replace(/[^A-Z0-9]/g, '');
}

/** UX F4: join a club with an invite code, then wait for approval. */
export default function JoinClubWizard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const errorText = useErrorText();
    const roleName = useRoleName();
    const { status, refresh } = useCollaboration();

    const [step, setStep] = useState(0);
    const [code, setCode] = useState('');
    const [folder, setFolder] = useState<FolderInfo | null>(null);
    const [hasLocalData, setHasLocalData] = useState(false);
    const [replaceLocal, setReplaceLocal] = useState(false);
    const [person, setPerson] = useState('');
    const [deviceName, setDeviceName] = useState('');
    const [password, setPassword] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const [safety, setSafety] = useState('');
    const [result, setResult] = useState<'member' | 'revoked' | null>(null);
    const [role, setRole] = useState('');

    const normalized = normalizeInviteCode(code);
    const codeError = !CODE_CHARS.test(normalized)
        ? t('collaboration.join.codeInvalid')
        : normalized.length > 0 && normalized.length < 24 ? t('collaboration.join.codeShort') : '';

    // Opened while a request is already pending: go straight to the waiting screen.
    useEffect(() => {
        if (status?.mode === 'joining' && !safety) {
            getSafetyCode().then((r) => {
                setSafety(r.safetyCode);
                setStep(3);
            }, () => {});
        }
    }, [status?.mode, safety]);

    useEffect(() => {
        getLocalData().then((r) => setHasLocalData(r.hasLocalData), () => {});
    }, []);

    // Wait for approval.
    useEffect(() => {
        if (step !== 3 || result) return;
        const poll = async () => {
            try {
                const st = await getStatus();
                if (st.mode === 'member') {
                    setRole(st.role ?? '');
                    setResult('member');
                    refresh();
                } else if (st.mode === 'revoked') {
                    setResult('revoked');
                }
            } catch {
                // try again
            }
        };
        const id = window.setInterval(poll, 5000);
        poll();
        return () => window.clearInterval(id);
    }, [step, result, refresh]);

    const submit = async () => {
        if (!folder) return;
        setBusy(true);
        setError('');
        try {
            const club = joinFolderClub(folder);
            const res = await joinClub({
                folder: club?.path ?? folder.path, code: normalized, person, deviceName, password,
                replaceLocalData: hasLocalData && replaceLocal,
            });
            setPassword('');
            setSafety(res.safetyCode);
            setStep(3);
            refresh();
        } catch (e) {
            setError(errorText(e));
        } finally {
            setBusy(false);
        }
    };

    const steps = ['code', 'storage', 'name', 'safety'].map((s) => t(`collaboration.join.steps.${s}`));

    return (
        <WizardFrame title={t('collaboration.join.title')} steps={steps} active={step}>
            {step === 0 && (
                <Stack spacing={2}>
                    <TextField
                        id="collab-invite-code"
                        label={t('collaboration.join.codeLabel')}
                        value={code}
                        onChange={(e) => setCode(e.target.value)}
                        placeholder="XXXX-XXXX-XXXX-XXXX-XXXX-XXXX"
                        inputProps={{ autoComplete: 'off', style: { fontFamily: 'ui-monospace, monospace', textTransform: 'uppercase', letterSpacing: '0.06em' } }}
                        error={!!codeError}
                        helperText={codeError || t('collaboration.join.codeHelp')}
                        autoFocus
                    />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => navigate(-1)}>{t('buttons.abort')}</Button>
                        <Button variant="contained" disabled={normalized.length !== 24 || !!codeError} onClick={() => setStep(1)}>
                            {t('collaboration.key.continue')}
                        </Button>
                    </Box>
                </Stack>
            )}

            {step === 1 && (
                <Stack spacing={2}>
                    <Typography variant="h6">{t('collaboration.join.storageQuestion')}</Typography>
                    <Typography color="text.secondary">{t('collaboration.join.storageIntro')}</Typography>
                    <FolderPicker purpose="join" onChange={setFolder} />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => setStep(0)}>{t('buttons.back', 'Zurück')}</Button>
                        <Button variant="contained" disabled={!folderUsable(folder, 'join')} onClick={() => setStep(2)}>
                            {t('collaboration.key.continue')}
                        </Button>
                    </Box>
                </Stack>
            )}

            {step === 2 && (
                <Stack spacing={2} component="form" onSubmit={(e: React.FormEvent) => { e.preventDefault(); submit(); }}>
                    <TextField id="collab-join-person" label={t('collaboration.create.person')} value={person}
                               onChange={(e) => setPerson(e.target.value)} required autoFocus />
                    <TextField id="collab-join-device" label={t('collaboration.create.deviceName')} value={deviceName}
                               onChange={(e) => setDeviceName(e.target.value)} helperText={t('collaboration.create.deviceHelp')} />
                    <TextField id="collab-join-password" type="password" label={t('collaboration.create.password')} value={password}
                               onChange={(e) => setPassword(e.target.value)} helperText={t('collaboration.create.passwordHelp')}
                               autoComplete="current-password" required />
                    {hasLocalData && (
                        <Alert severity="warning">
                            <Typography sx={{ fontWeight: 600 }}>{t('collaboration.join.localDataTitle')}</Typography>
                            <Typography variant="body2">{t('collaboration.join.localDataText')}</Typography>
                            <FormControlLabel
                                control={<Checkbox id="collab-join-replace" checked={replaceLocal} onChange={(e) => setReplaceLocal(e.target.checked)} />}
                                label={t('collaboration.join.localDataConfirm')}
                            />
                        </Alert>
                    )}
                    {error && <Alert severity="error">{error}</Alert>}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => setStep(1)} disabled={busy}>{t('buttons.back', 'Zurück')}</Button>
                        <Button type="submit" variant="contained"
                                disabled={busy || !person.trim() || !password || (hasLocalData && !replaceLocal)}>
                            {busy ? <CircularProgress size={18} /> : t('collaboration.join.submit')}
                        </Button>
                    </Box>
                </Stack>
            )}

            {step === 3 && !result && (
                <Stack spacing={2}>
                    <Typography variant="h6">{t('collaboration.join.safetyTitle')}</Typography>
                    <Typography>{t('collaboration.join.safetyText')}</Typography>
                    {safety && <SafetyCodeDisplay code={safety} />}
                    <Typography>{t('collaboration.join.safetyWait')}</Typography>
                    <Alert severity="info" icon={<CircularProgress size={18} />}>{t('collaboration.join.waiting')}</Alert>
                </Stack>
            )}

            {result === 'member' && (
                <Stack spacing={2}>
                    <Alert severity="success">{t('collaboration.join.welcome', { role: roleName(role) })}</Alert>
                    <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button variant="contained" onClick={() => navigate('/dashboard')}>{t('collaboration.join.toApp')}</Button>
                    </Box>
                </Stack>
            )}
            {result === 'revoked' && <Alert severity="error">{t('collaboration.join.rejected')}</Alert>}
        </WizardFrame>
    );
}
