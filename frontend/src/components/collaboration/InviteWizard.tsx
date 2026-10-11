import React, { useMemo, useState } from 'react';
import {
    Alert, Box, Button, Checkbox, FormControlLabel, Stack, Step, StepLabel, Stepper, Tab, Tabs, TextField, Typography,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { ClubInvite, ClubRole, createInvite } from '../../api/collaboration';
import { useCollaboration } from '../../context/CollaborationContext';
import { ChoiceCard, copyText, PermissionRow, useErrorText, useRoleName } from './common';

const PROVIDERS = ['Nextcloud', 'OneDrive', 'Dropbox', 'Google Drive', 'other'] as const;

// Providers' own help on sharing folders (UX decision: text steps plus a link,
// no screenshots that go out of date).
const HELP_LINKS: Record<string, string> = {
    Nextcloud: 'https://docs.nextcloud.com/server/latest/user_manual/en/files/sharing.html',
    OneDrive: 'https://support.microsoft.com/onedrive',
    Dropbox: 'https://help.dropbox.com/share',
    'Google Drive': 'https://support.google.com/drive/answer/7166529',
};

function providerFromPath(path?: string): (typeof PROVIDERS)[number] {
    const p = (path ?? '').toLowerCase();
    if (p.includes('nextcloud') || p.includes('owncloud')) return 'Nextcloud';
    if (p.includes('onedrive')) return 'OneDrive';
    if (p.includes('dropbox')) return 'Dropbox';
    if (p.includes('google')) return 'Google Drive';
    return 'other';
}

type Props = {
    roles: ClubRole[];
    onDone: (invite?: ClubInvite) => void;
};

/** UX F3: who and which role, folder access at the provider, invite code. */
export default function InviteWizard({ roles, onDone }: Props) {
    const { t } = useTranslation();
    const errorText = useErrorText();
    const roleName = useRoleName();
    const { status } = useCollaboration();

    const [step, setStep] = useState(0);
    const [name, setName] = useState('');
    const [role, setRole] = useState<string | null>(null);
    const [provider, setProvider] = useState<(typeof PROVIDERS)[number]>(providerFromPath(status?.storagePath));
    const [shared, setShared] = useState(false);
    const [invite, setInvite] = useState<ClubInvite | null>(null);
    const [showMail, setShowMail] = useState(false);
    const [error, setError] = useState('');
    const [copyMsg, setCopyMsg] = useState('');
    const [busy, setBusy] = useState(false);

    const first = name.trim().split(/\s+/)[0] || t('collaboration.invite.thePerson');
    const selectedRole = roles.find((r) => r.id === role);
    const guide = t(`collaboration.invite.guides.${provider}`, { returnObjects: true, name: first }) as string[];

    const mail = useMemo(() => invite ? t('collaboration.invite.mail', {
        name: first, role: roleName(invite.role), club: status?.clubName ?? '', code: invite.code, sender: status?.person ?? '',
    }) : '', [invite, first, roleName, status, t]);

    const create = async () => {
        if (!role) return;
        setBusy(true);
        setError('');
        try {
            setInvite(await createInvite(name.trim(), role));
            setStep(2);
        } catch (e) {
            setError(errorText(e));
        } finally {
            setBusy(false);
        }
    };

    const copy = async (text: string) => setCopyMsg((await copyText(text)) ? t('collaboration.invite.copied') : t('collaboration.invite.copyFailed'));

    const steps = ['who', 'access', 'code'].map((s) => t(`collaboration.invite.steps.${s}`));

    return (
        <Stack spacing={2.5}>
            <Stepper activeStep={step} alternativeLabel>
                {steps.map((s) => <Step key={s}><StepLabel>{s}</StepLabel></Step>)}
            </Stepper>

            {step === 0 && (
                <>
                    <TextField id="collab-invite-name" label={t('collaboration.invite.name')} placeholder={t('collaboration.invite.namePlaceholder')}
                               value={name} onChange={(e) => setName(e.target.value)} autoFocus required />
                    <Typography sx={{ fontWeight: 600 }}>{t('collaboration.invite.role')}</Typography>
                    <Stack spacing={1}>
                        {roles.map((r) => (
                            <ChoiceCard key={r.id} selected={role === r.id} onSelect={() => setRole(r.id)}
                                        title={roleName(r.id)} description={t(`collaboration.roleDesc.${r.id}`, { defaultValue: '' })} />
                        ))}
                    </Stack>
                    {selectedRole && (
                        <Box>
                            <Typography variant="body2" color="text.secondary">
                                {t('collaboration.invite.whatRole', { name: first, role: roleName(selectedRole.id) })}
                            </Typography>
                            <PermissionRow permissions={selectedRole.permissions} />
                        </Box>
                    )}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => onDone()}>{t('buttons.abort')}</Button>
                        <Button variant="contained" disabled={!name.trim() || !role} onClick={() => setStep(1)}>{t('buttons.next')}</Button>
                    </Box>
                </>
            )}

            {step === 1 && (
                <>
                    <Typography variant="h6">{t('collaboration.invite.accessTitle', { name: first })}</Typography>
                    <Typography>{t('collaboration.invite.accessIntro')}</Typography>
                    {status?.storagePath && (
                        <Typography variant="body2" sx={{ wordBreak: 'break-all' }}>
                            {t('collaboration.invite.accessFolder', { path: status.storagePath })}
                        </Typography>
                    )}
                    <Tabs value={provider} onChange={(_, v) => setProvider(v)} variant="scrollable" allowScrollButtonsMobile>
                        {PROVIDERS.map((p) => (
                            <Tab key={p} value={p} label={p === 'other' ? t('collaboration.invite.otherProvider') : p} />
                        ))}
                    </Tabs>
                    <Box component="ol" sx={{ m: 0, pl: 3, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                        {guide.map((line) => <li key={line}><Typography>{line}</Typography></li>)}
                    </Box>
                    {HELP_LINKS[provider] && (
                        <Button href={HELP_LINKS[provider]} target="_blank" rel="noreferrer" sx={{ alignSelf: 'flex-start' }}>
                            {t('collaboration.invite.providerHelp', { provider })} ↗
                        </Button>
                    )}
                    <FormControlLabel
                        control={<Checkbox id="collab-invite-shared" checked={shared} onChange={(e) => setShared(e.target.checked)} />}
                        label={t('collaboration.invite.accessDone', { name: first })}
                    />
                    {error && <Alert severity="error">{error}</Alert>}
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Button onClick={() => setStep(0)} disabled={busy}>{t('buttons.back')}</Button>
                        <Button variant="contained" disabled={!shared || busy} onClick={create}>{t('buttons.next')}</Button>
                    </Box>
                </>
            )}

            {step === 2 && invite && (
                <>
                    <Typography variant="h6">{t('collaboration.invite.codeTitle', { name: invite.person, role: roleName(invite.role) })}</Typography>
                    <Typography
                        sx={{
                            fontFamily: '"JetBrains Mono", ui-monospace, monospace', fontWeight: 700, fontSize: { xs: 16, sm: 20 },
                            textAlign: 'center', p: 1.5, borderRadius: 2, bgcolor: 'action.hover', wordBreak: 'break-all', userSelect: 'all',
                        }}
                    >
                        {invite.code}
                    </Typography>
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                        <Button variant="outlined" onClick={() => copy(invite.code ?? '')}>{t('collaboration.invite.copyCode')}</Button>
                        <Button variant="outlined" onClick={() => setShowMail((s) => !s)}>
                            {showMail ? t('collaboration.invite.hideMail') : t('collaboration.invite.showMail')}
                        </Button>
                        {copyMsg && <Typography variant="body2" sx={{ alignSelf: 'center' }}>{copyMsg}</Typography>}
                    </Box>
                    {showMail && (
                        <>
                            <Box component="pre" sx={{ m: 0, p: 1.5, borderRadius: 2, bgcolor: 'action.hover', whiteSpace: 'pre-wrap', fontFamily: 'inherit' }}>
                                {mail}
                            </Box>
                            <Button variant="outlined" size="small" sx={{ alignSelf: 'flex-start' }} onClick={() => copy(mail)}>
                                {t('collaboration.invite.copyMail')}
                            </Button>
                        </>
                    )}
                    <Typography>{t('collaboration.invite.codeHint', { name: first })}</Typography>
                    <Alert severity="warning">{t('collaboration.invite.safetyHint', { name: first })}</Alert>
                    <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                        <Button variant="contained" onClick={() => onDone(invite)}>{t('collaboration.invite.done')}</Button>
                    </Box>
                </>
            )}
        </Stack>
    );
}
