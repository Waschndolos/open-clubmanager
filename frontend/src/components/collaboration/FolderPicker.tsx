import React, { useEffect, useState } from 'react';
import { Alert, Box, Button, CircularProgress, Stack, TextField, Typography } from '@mui/material';
import { useTranslation } from 'react-i18next';
import { checkFolder, CollaborationError, FolderInfo, listFolders } from '../../api/collaboration';
import { ChoiceCard } from './common';

type Props = {
    /** create: the club gets a subfolder here; join: the folder must hold a club. */
    purpose: 'create' | 'join';
    clubName?: string;
    onChange: (folder: FolderInfo | null) => void;
};

/** Whether a checked folder can be used for the purpose. */
export function folderUsable(f: FolderInfo | null, purpose: 'create' | 'join'): boolean {
    if (!f || !f.writable) return false;
    if (purpose === 'create') return !f.club;
    return !!f.club || f.clubsInside.length === 1;
}

/** Folder of the club that joining will use: the folder itself or its only club subfolder. */
export function joinFolderClub(f: FolderInfo) {
    return f.club ?? (f.clubsInside.length === 1 ? f.clubsInside[0] : undefined);
}

function FolderNotes({ folder, purpose, clubName }: { folder: FolderInfo; purpose: Props['purpose']; clubName?: string }) {
    const { t } = useTranslation();
    if (!folder.writable) return <Alert severity="error">{t('collaboration.folder.notWritable')}</Alert>;
    const kind =
        folder.kind === 'sync' ? <Alert severity="success">{t('collaboration.folder.kindSync', { provider: folder.provider })}</Alert>
        : folder.kind === 'network' ? <Alert severity="info">{t('collaboration.folder.kindNetwork')}</Alert>
        : <Alert severity="warning">{t('collaboration.folder.kindLocal')}</Alert>;
    if (purpose === 'create') {
        const existing = folder.club ?? folder.clubsInside[0];
        return (
            <Stack spacing={1}>
                {kind}
                {folder.club
                    ? <Alert severity="warning">{t('collaboration.folder.clubExists', { name: folder.club.name })}</Alert>
                    : <>
                        {existing && <Alert severity="info">{t('collaboration.folder.clubExists', { name: existing.name })}</Alert>}
                        {clubName && <Typography variant="body2" color="text.secondary">{t('collaboration.folder.subfolder', { name: clubName })}</Typography>}
                    </>}
            </Stack>
        );
    }
    const club = joinFolderClub(folder);
    return (
        <Stack spacing={1}>
            {kind}
            {club
                ? <Alert severity="success">{t('collaboration.folder.containsClub', { name: club.name })}</Alert>
                : folder.clubsInside.length > 1
                    ? <Alert severity="warning">{t('collaboration.folder.severalClubs')}</Alert>
                    : <Alert severity="warning">{t('collaboration.folder.noClub')}</Alert>}
        </Stack>
    );
}

/** Lists detected sync folders and lets the user enter any folder or network path (UX F2 step 1, F4 step 2). */
export default function FolderPicker({ purpose, clubName, onChange }: Props) {
    const { t } = useTranslation();
    const [detected, setDetected] = useState<FolderInfo[] | null>(null);
    const [selected, setSelected] = useState<string | 'custom' | null>(null);
    const [customPath, setCustomPath] = useState('');
    const [custom, setCustom] = useState<FolderInfo | null>(null);
    const [customError, setCustomError] = useState('');
    const [checking, setChecking] = useState(false);

    useEffect(() => {
        listFolders().then(setDetected, () => setDetected([]));
    }, []);

    // When joining, prefer folders that already contain a club.
    const suggestions = (detected ?? [])
        .flatMap((f) => (purpose === 'join' ? [...f.clubsInside.map((c) => ({ ...f, path: c.path, club: c, clubsInside: [] })), f] : [f]))
        .filter((f, i, all) => all.findIndex((g) => g.path === f.path) === i);

    const selectDetected = (f: FolderInfo) => {
        setSelected(f.path);
        onChange(f);
    };

    const runCheck = async () => {
        const path = customPath.trim();
        if (!path) return;
        setChecking(true);
        setCustomError('');
        try {
            const info = await checkFolder(path);
            setCustom(info);
            onChange(info);
        } catch (e) {
            setCustom(null);
            onChange(null);
            setCustomError(e instanceof CollaborationError && e.code === 'FOLDER_MISSING'
                ? t('collaboration.folder.missing')
                : t('collaboration.errors.default', { message: e instanceof Error ? e.message : String(e) }));
        } finally {
            setChecking(false);
        }
    };

    const current = selected === 'custom' ? custom : suggestions.find((f) => f.path === selected) ?? null;

    return (
        <Stack spacing={1.25}>
            <Typography variant="body2" color="text.secondary">{t('collaboration.folder.detected')}</Typography>
            {detected === null && <CircularProgress size={20} />}
            {detected !== null && suggestions.length === 0 && (
                <Typography variant="body2" color="text.secondary">{t('collaboration.folder.none')}</Typography>
            )}
            {suggestions.map((f) => (
                <ChoiceCard
                    key={f.path}
                    selected={selected === f.path}
                    onSelect={() => selectDetected(f)}
                    title={f.path}
                    description={purpose === 'join' && f.club ? t('collaboration.folder.containsClub', { name: f.club.name }) : f.provider}
                />
            ))}
            <ChoiceCard
                selected={selected === 'custom'}
                onSelect={() => {
                    setSelected('custom');
                    onChange(custom);
                }}
                title={t('collaboration.folder.custom')}
                description={t('collaboration.folder.customHint')}
            >
                {selected === 'custom' && (
                    <Box sx={{ display: 'flex', gap: 1, mt: 1, flexWrap: 'wrap' }} onClick={(e) => e.stopPropagation()}>
                        <TextField
                            id="collab-folder-path"
                            size="small"
                            label={t('collaboration.folder.pathLabel')}
                            value={customPath}
                            onChange={(e) => {
                                setCustomPath(e.target.value);
                                setCustom(null);
                                setCustomError('');
                                onChange(null);
                            }}
                            onBlur={runCheck}
                            onKeyDown={(e) => {
                                e.stopPropagation();
                                if (e.key === 'Enter') runCheck();
                            }}
                            helperText={customError || t('collaboration.folder.pathHelp')}
                            error={!!customError}
                            sx={{ flex: 1, minWidth: 220 }}
                            autoFocus
                        />
                        <Button variant="outlined" onClick={runCheck} disabled={checking || !customPath.trim()} sx={{ alignSelf: 'flex-start', mt: 0.25 }}>
                            {checking ? t('collaboration.folder.checking') : t('collaboration.folder.check')}
                        </Button>
                    </Box>
                )}
            </ChoiceCard>
            {current && <FolderNotes folder={current} purpose={purpose} clubName={clubName} />}
        </Stack>
    );
}
