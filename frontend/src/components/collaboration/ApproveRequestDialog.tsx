import React, { useState } from 'react';
import { Alert, Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { approveRequest, ClubJoinRequest, CollaborationError, rejectRequest } from '../../api/collaboration';
import { useErrorText, useRoleName } from './common';

type Props = {
    request: ClubJoinRequest | null;
    onClose: (result?: 'approved' | 'rejected') => void;
};

/**
 * UX F5: the admin types the safety code the new member reads out. Typing
 * forces a real comparison; two codes side by side are easily clicked away.
 */
export default function ApproveRequestDialog({ request, onClose }: Props) {
    const { t } = useTranslation();
    const errorText = useErrorText();
    const roleName = useRoleName();
    const [code, setCode] = useState('');
    const [error, setError] = useState('');
    const [busy, setBusy] = useState(false);

    const close = (result?: 'approved' | 'rejected') => {
        setCode('');
        setError('');
        onClose(result);
    };

    if (!request) return null;
    const name = request.person;
    const digits = code.replace(/\D/g, '');
    const invalid = request.error;

    const approve = async () => {
        setBusy(true);
        setError('');
        try {
            await approveRequest(request.deviceId, digits);
            close('approved');
        } catch (e) {
            if (e instanceof CollaborationError && e.code === 'WRONG_SAFETY_CODE') {
                setError(t('collaboration.request.wrongCode'));
                setCode('');
            } else if (e instanceof CollaborationError && e.code === 'TOO_MANY_ATTEMPTS') {
                setError(t('collaboration.request.tooMany'));
                setTimeout(() => close('rejected'), 2500);
            } else {
                setError(errorText(e));
            }
        } finally {
            setBusy(false);
        }
    };

    const reject = async () => {
        setBusy(true);
        try {
            await rejectRequest(request.deviceId);
            close('rejected');
        } catch (e) {
            setError(errorText(e));
        } finally {
            setBusy(false);
        }
    };

    return (
        <Dialog open onClose={() => close()} maxWidth="xs" fullWidth>
            <DialogTitle>{t('collaboration.request.title')}</DialogTitle>
            <DialogContent>
                <Stack spacing={2}>
                    <div>
                        <Typography sx={{ fontWeight: 600 }}>
                            {name}{request.deviceName ? ` – ${request.deviceName}` : ''}
                        </Typography>
                        {request.role && (
                            <Typography variant="body2" color="text.secondary">
                                {t('collaboration.request.invitedAs', { role: roleName(request.role) })}
                            </Typography>
                        )}
                    </div>
                    {invalid ? (
                        <Alert severity="warning">
                            {t(`collaboration.request.invalid.${invalid}`, { defaultValue: t('collaboration.request.invalid.default') })}
                        </Alert>
                    ) : (
                        <>
                            <Typography>{t('collaboration.request.askCode', { name })}</Typography>
                            <TextField
                                id="collab-safety-input"
                                value={code}
                                onChange={(e) => setCode(e.target.value.replace(/[^\d ]/g, '').slice(0, 7))}
                                placeholder="000 000"
                                inputProps={{
                                    inputMode: 'numeric', autoComplete: 'off',
                                    style: { fontFamily: 'ui-monospace, monospace', fontSize: 24, textAlign: 'center', letterSpacing: '0.1em' },
                                }}
                                onKeyDown={(e) => { if (e.key === 'Enter' && digits.length === 6) approve(); }}
                                sx={{ maxWidth: 200, alignSelf: 'center' }}
                                autoFocus
                            />
                            <Typography variant="body2" color="text.secondary">{t('collaboration.request.mismatchHint')}</Typography>
                        </>
                    )}
                    {error && <Alert severity="error">{error}</Alert>}
                </Stack>
            </DialogContent>
            <DialogActions>
                <Button color="error" onClick={reject} disabled={busy}>{t('collaboration.request.reject')}</Button>
                <Button onClick={() => close()} disabled={busy}>{t('collaboration.request.later')}</Button>
                {!invalid && (
                    <Button variant="contained" onClick={approve} disabled={busy || digits.length !== 6}>
                        {t('collaboration.request.approve')}
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
}
