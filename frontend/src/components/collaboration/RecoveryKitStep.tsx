import React, { useMemo, useState } from 'react';
import { Alert, Box, Button, Checkbox, FormControlLabel, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { copyText } from './common';

type Props = {
    kit: string;
    clubName: string;
    onDone: () => void;
};

function printText(title: string, text: string) {
    const frame = document.createElement('iframe');
    frame.style.position = 'fixed';
    frame.style.width = '0';
    frame.style.height = '0';
    frame.style.border = '0';
    document.body.appendChild(frame);
    const doc = frame.contentDocument;
    if (!doc) return;
    doc.open();
    doc.write('<!doctype html><title></title><pre style="font: 13px/1.6 ui-monospace, monospace; white-space: pre-wrap"></pre>');
    doc.close();
    doc.title = title;
    doc.querySelector('pre')!.textContent = text;
    frame.contentWindow?.focus();
    frame.contentWindow?.print();
    setTimeout(() => frame.remove(), 1000);
}

function saveText(fileName: string, text: string) {
    const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }));
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
}

/** Shows the emergency key once and makes the user confirm they kept it (UX F2 step 3). */
export default function RecoveryKitStep({ kit, clubName, onDone }: Props) {
    const { t } = useTranslation();
    const [check, setCheck] = useState('');
    const [kept, setKept] = useState(false);
    const [copied, setCopied] = useState<boolean | null>(null);

    const expected = useMemo(() => {
        const line = kit.split('\n').find((l) => l.trim().startsWith('check:'));
        const value = line ? line.split(':')[1].trim() : '';
        return value.slice(-4).toLowerCase();
    }, [kit]);
    const checkOk = check.trim().toLowerCase() === expected;
    const fileName = `Notfallschluessel-${clubName.replace(/[^\p{L}\p{N}]+/gu, '-')}.txt`;

    return (
        <Stack spacing={2}>
            <Typography variant="h6">{t('collaboration.key.title')}</Typography>
            <Typography>{t('collaboration.key.intro')}</Typography>
            <Box
                component="pre"
                sx={{
                    m: 0, p: 1.5, borderRadius: 2, bgcolor: 'action.hover', overflowX: 'auto',
                    fontFamily: '"JetBrains Mono", ui-monospace, monospace', fontSize: 12.5, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
                }}
            >
                {kit}
            </Box>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                <Button variant="outlined" onClick={() => printText(t('collaboration.key.title'), kit)}>{t('collaboration.key.print')}</Button>
                <Button variant="outlined" onClick={() => saveText(fileName, kit)}>{t('collaboration.key.save')}</Button>
                <Button variant="outlined" onClick={async () => setCopied(await copyText(kit))}>{t('collaboration.key.copy')}</Button>
            </Box>
            {copied === false && <Alert severity="warning">{t('collaboration.invite.copyFailed')}</Alert>}
            <Alert severity="warning">{t('collaboration.key.keepSafe')}</Alert>
            <TextField
                id="collab-key-check"
                label={t('collaboration.key.check')}
                value={check}
                onChange={(e) => setCheck(e.target.value)}
                inputProps={{ maxLength: 4, autoComplete: 'off', style: { fontFamily: 'ui-monospace, monospace' } }}
                error={check.length === 4 && !checkOk}
                helperText={check.length === 4 && !checkOk ? t('collaboration.key.checkError') : ' '}
                sx={{ maxWidth: 360 }}
            />
            <FormControlLabel
                control={<Checkbox id="collab-key-kept" checked={kept} onChange={(e) => setKept(e.target.checked)} />}
                label={t('collaboration.key.confirm')}
            />
            <Typography variant="body2" color="text.secondary">{t('collaboration.key.deleted')}</Typography>
            <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                <Button variant="contained" disabled={!checkOk || !kept} onClick={onDone}>{t('collaboration.key.continue')}</Button>
            </Box>
        </Stack>
    );
}
