import React, { useEffect, useState } from 'react';
import { Alert, Box, Button, Checkbox, CircularProgress, FormControlLabel, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { FolderInfo, getLocalData, recoverClub } from '../../api/collaboration';
import FolderPicker, { folderUsable, joinFolderClub } from '../../components/collaboration/FolderPicker';
import { useErrorText } from '../../components/collaboration/common';
import { useCollaboration } from '../../context/CollaborationContext';
import WizardFrame from './WizardFrame';

/** UX F12: use the emergency key to become an admin device. */
export default function RecoverWizard() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const errorText = useErrorText();
    const { status, refresh } = useCollaboration();
    const needsFolder = !status || status.mode === 'local';

    const [kit, setKit] = useState('');
    const [folder, setFolder] = useState<FolderInfo | null>(null);
    const [hasLocalData, setHasLocalData] = useState(false);
    const [replaceLocal, setReplaceLocal] = useState(false);
    const [person, setPerson] = useState(status?.person ?? '');
    const [deviceName, setDeviceName] = useState(status?.deviceName ?? '');
    const [password, setPassword] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const [done, setDone] = useState(false);

    useEffect(() => {
        if (needsFolder) getLocalData().then((r) => setHasLocalData(r.hasLocalData), () => {});
    }, [needsFolder]);

    const submit = async () => {
        setBusy(true);
        setError('');
        try {
            const club = folder ? joinFolderClub(folder) : undefined;
            await recoverClub({
                folder: needsFolder ? club?.path ?? folder?.path : undefined,
                recoveryKit: kit, person, deviceName, password,
                replaceLocalData: hasLocalData && replaceLocal,
            });
            setKit('');
            setPassword('');
            setDone(true);
            refresh();
        } catch (e) {
            setError(errorText(e));
        } finally {
            setBusy(false);
        }
    };

    if (done) {
        return (
            <WizardFrame title={t('collaboration.recover.title')}>
                <Alert severity="success">{t('collaboration.recover.done')}</Alert>
                <Typography>{t('collaboration.recover.reminder')}</Typography>
                <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                    <Button variant="contained" onClick={() => navigate('/collaboration?tab=people')}>{t('collaboration.recover.toPeople')}</Button>
                </Box>
            </WizardFrame>
        );
    }

    const ready = kit.trim() && person.trim() && password && (!needsFolder || folderUsable(folder, 'join')) && (!hasLocalData || replaceLocal);

    return (
        <WizardFrame title={t('collaboration.recover.title')}>
            <Stack spacing={2} component="form" onSubmit={(e: React.FormEvent) => { e.preventDefault(); submit(); }}>
                <Alert severity="warning">{t('collaboration.recover.intro')}</Alert>
                <TextField id="collab-recover-kit" label={t('collaboration.recover.kitLabel')} value={kit}
                           onChange={(e) => setKit(e.target.value)} helperText={t('collaboration.recover.kitHelp')}
                           multiline minRows={6} inputProps={{ style: { fontFamily: 'ui-monospace, monospace', fontSize: 13 }, autoComplete: 'off', spellCheck: false }} />
                {needsFolder && (
                    <>
                        <Typography variant="h6">{t('collaboration.join.storageQuestion')}</Typography>
                        <FolderPicker purpose="join" onChange={setFolder} />
                    </>
                )}
                <TextField id="collab-recover-person" label={t('collaboration.create.person')} value={person}
                           onChange={(e) => setPerson(e.target.value)} required />
                <TextField id="collab-recover-device" label={t('collaboration.create.deviceName')} value={deviceName}
                           onChange={(e) => setDeviceName(e.target.value)} />
                <TextField id="collab-recover-password" type="password" label={t('collaboration.create.password')} value={password}
                           onChange={(e) => setPassword(e.target.value)} helperText={t('collaboration.create.passwordHelp')}
                           autoComplete="current-password" required />
                {needsFolder && hasLocalData && (
                    <Alert severity="warning">
                        <Typography sx={{ fontWeight: 600 }}>{t('collaboration.join.localDataTitle')}</Typography>
                        <Typography variant="body2">{t('collaboration.join.localDataText')}</Typography>
                        <FormControlLabel
                            control={<Checkbox id="collab-recover-replace" checked={replaceLocal} onChange={(e) => setReplaceLocal(e.target.checked)} />}
                            label={t('collaboration.join.localDataConfirm')}
                        />
                    </Alert>
                )}
                {error && <Alert severity="error">{error}</Alert>}
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Button onClick={() => navigate(-1)} disabled={busy}>{t('buttons.abort')}</Button>
                    <Button type="submit" variant="contained" disabled={busy || !ready}>
                        {busy ? <CircularProgress size={18} /> : t('collaboration.recover.submit')}
                    </Button>
                </Box>
            </Stack>
        </WizardFrame>
    );
}
