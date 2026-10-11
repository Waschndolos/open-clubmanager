import React, { useState } from 'react';
import { Alert, Box, Button, LinearProgress, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { createClub, FolderInfo } from '../../api/collaboration';
import FolderPicker, { folderUsable } from '../../components/collaboration/FolderPicker';
import RecoveryKitStep from '../../components/collaboration/RecoveryKitStep';
import { useErrorText } from '../../components/collaboration/common';
import { useCollaboration } from '../../context/CollaborationContext';
import WizardFrame from './WizardFrame';

/** UX F2: set up a club for collaboration. */
export default function CreateClubWizard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const errorText = useErrorText();
    const { refresh } = useCollaboration();

    const [step, setStep] = useState(0);
    const [folder, setFolder] = useState<FolderInfo | null>(null);
    const [clubName, setClubName] = useState('');
    const [person, setPerson] = useState('');
    const [deviceName, setDeviceName] = useState('');
    const [password, setPassword] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const [kit, setKit] = useState('');

    const steps = ['storage', 'club', 'key', 'done'].map((s) => t(`collaboration.create.steps.${s}`));

    const submit = async () => {
        if (!folder) return;
        setBusy(true);
        setError('');
        try {
            const res = await createClub({ folder: folder.path, clubName, person, deviceName, password });
            setPassword('');
            setKit(res.recoveryKit);
            setStep(2);
            refresh();
        } catch (e) {
            setError(errorText(e));
        } finally {
            setBusy(false);
        }
    };

    return (
        <WizardFrame title={t('collaboration.create.title')} steps={steps} active={step}>
            {step === 0 && (
                <Stack spacing={2}>
                    <Typography variant="h6">{t('collaboration.create.storageQuestion')}</Typography>
                    <Typography color="text.secondary">{t('collaboration.create.storageIntro')}</Typography>
                    <FolderPicker purpose="create" clubName={clubName || undefined} onChange={setFolder} />
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => navigate(-1)}>{t('buttons.abort')}</Button>
                        <Button variant="contained" disabled={!folderUsable(folder, 'create')} onClick={() => setStep(1)}>
                            {t('collaboration.key.continue')}
                        </Button>
                    </Box>
                </Stack>
            )}

            {step === 1 && (
                <Stack spacing={2} component="form" onSubmit={(e: React.FormEvent) => { e.preventDefault(); submit(); }}>
                    <TextField id="collab-club-name" label={t('collaboration.create.clubName')} value={clubName}
                               onChange={(e) => setClubName(e.target.value)} required autoFocus />
                    <TextField id="collab-person" label={t('collaboration.create.person')} value={person}
                               onChange={(e) => setPerson(e.target.value)} required />
                    <TextField id="collab-device" label={t('collaboration.create.deviceName')} value={deviceName}
                               onChange={(e) => setDeviceName(e.target.value)} helperText={t('collaboration.create.deviceHelp')} />
                    <TextField id="collab-password" type="password" label={t('collaboration.create.password')} value={password}
                               onChange={(e) => setPassword(e.target.value)} helperText={t('collaboration.create.passwordHelp')}
                               autoComplete="current-password" required />
                    <Alert severity="info">{t('collaboration.create.roleInfo')} {t('collaboration.create.localData')}</Alert>
                    {folder && <Typography variant="body2" color="text.secondary">{t('collaboration.folder.subfolder', { name: clubName || '…' })}</Typography>}
                    {busy && <><Typography>{t('collaboration.create.creating')}</Typography><LinearProgress /></>}
                    {error && <Alert severity="error">{error}</Alert>}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => setStep(0)} disabled={busy}>{t('buttons.back', 'Zurück')}</Button>
                        <Button type="submit" variant="contained" disabled={busy || !clubName.trim() || !person.trim() || !password}>
                            {t('collaboration.create.submit')}
                        </Button>
                    </Box>
                </Stack>
            )}

            {step === 2 && (
                <RecoveryKitStep
                    kit={kit}
                    clubName={clubName}
                    onDone={() => {
                        setKit('');
                        setStep(3);
                    }}
                />
            )}

            {step === 3 && (
                <Stack spacing={2}>
                    <Typography variant="h6">{t('collaboration.create.doneTitle')}</Typography>
                    <Typography>{t('collaboration.create.doneText')}</Typography>
                    <Box sx={{ display: 'flex', gap: 1, justifyContent: 'flex-end', flexWrap: 'wrap' }}>
                        <Button onClick={() => navigate('/dashboard')}>{t('collaboration.create.toOverview')}</Button>
                        <Button variant="contained" onClick={() => navigate('/collaboration?tab=invites&new=1')}>
                            {t('collaboration.create.invite')}
                        </Button>
                    </Box>
                </Stack>
            )}
        </WizardFrame>
    );
}
