import React, { useCallback, useEffect, useState } from 'react';
import { Alert, Box, Button, CircularProgress, Paper, Stack, TextField, Typography } from '@mui/material';
import ReportProblemOutlinedIcon from '@mui/icons-material/ReportProblemOutlined';
import { useTranslation } from 'react-i18next';
import { listConflicts, SyncConflict } from '../../api/collaboration';
import { fetchMembers, updateMember } from '../../api/members';
import { fetchGroups, updateGroup } from '../../api/groups';
import { fetchRoles, updateRole } from '../../api/roles';
import { fetchSections, updateSection } from '../../api/sections';
import { Member, NamedArtifact } from '../../api/types';
import PageHeader from '../../components/common/PageHeader';
import { useErrorText } from '../../components/collaboration/common';
import { useCollaboration, usePermission } from '../../context/CollaborationContext';

type Named = NamedArtifact & { members?: Member[] };

const namedApis = {
    groups: { fetch: fetchGroups, update: updateGroup },
    roles: { fetch: fetchRoles, update: updateRole },
    sections: { fetch: fetchSections, update: updateSection },
} as const;

/** UX F8: duplicate unique values after synchronizing (spec §14.2). */
export default function Conflicts() {
    const { t } = useTranslation();
    const errorText = useErrorText();
    const { refresh } = useCollaboration();
    const canCore = usePermission('membersCore') === 'write';
    const canContact = usePermission('membersContact') === 'write';

    const [conflicts, setConflicts] = useState<SyncConflict[] | null>(null);
    const [members, setMembers] = useState<Member[]>([]);
    const [named, setNamed] = useState<Record<string, Named[]>>({});
    const [input, setInput] = useState<Record<string, string>>({});
    const [error, setError] = useState('');

    const load = useCallback(async () => {
        try {
            const list = await listConflicts();
            setConflicts(list);
            const tables = new Set(list.map((c) => c.table));
            if (tables.has('members') || tables.has('member_contacts')) setMembers(await fetchMembers());
            const next: Record<string, Named[]> = {};
            for (const table of ['groups', 'roles', 'sections'] as const) {
                if (tables.has(table)) next[table] = await namedApis[table].fetch();
            }
            setNamed(next);
        } catch (e) {
            setError(errorText(e));
        }
    }, []);

    useEffect(() => { load(); }, [load]);

    const memberLabel = (id: string) => {
        const m = members.find((x) => x.id === id);
        return m ? [m.firstName, m.lastName].filter(Boolean).join(' ') || m.email : t('collaboration.conflicts.unknownRow', { id: id.slice(0, 8) });
    };
    const nextFreeNumber = () => Math.max(0, ...members.map((m) => m.number ?? 0)) + 1;

    const save = async (c: SyncConflict, value: string) => {
        setError('');
        try {
            if (c.table === 'members' || c.table === 'member_contacts') {
                const m = members.find((x) => x.id === c.rowId);
                if (!m) return;
                await updateMember(c.table === 'members' ? { ...m, number: Number(value) } : { ...m, email: value });
            } else {
                const api = namedApis[c.table as keyof typeof namedApis];
                const row = named[c.table]?.find((x) => x.id === c.rowId);
                if (!api || !row) return;
                await api.update({ ...row, name: value } as never);
            }
            await load();
            refresh();
        } catch (e) {
            setError(errorText(e));
        }
    };

    const key = (c: SyncConflict) => `${c.table}:${c.rowId}:${c.column}`;

    const card = (c: SyncConflict) => {
        const isMember = c.table === 'members' || c.table === 'member_contacts';
        const canWrite = c.table === 'member_contacts' ? canContact : canCore;
        const title = c.column === 'number' ? t('collaboration.conflicts.numberTitle', { value: c.value })
            : c.column === 'email' ? t('collaboration.conflicts.emailTitle', { value: c.value })
            : t('collaboration.conflicts.nameTitle', { value: c.value });
        const label = (id: string) => isMember ? memberLabel(id) : `${t(`collaboration.conflicts.tables.${c.table}`)} „${c.value}“`;
        const k = key(c);
        return (
            <Paper key={k} sx={{ p: 2.5, display: 'flex', flexDirection: 'column', gap: 1.5 }}>
                <Typography variant="h6" component="h2">{title}</Typography>
                <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 1.5 }}>
                    <Box sx={{ border: 1, borderColor: 'divider', borderRadius: 2, p: 1.5 }}>
                        <Typography sx={{ fontWeight: 600 }}>{label(c.keptRowId)}</Typography>
                        <Typography variant="body2" color="success.main">✓ {t('collaboration.conflicts.keeps')}</Typography>
                    </Box>
                    <Box sx={{ border: 1, borderColor: 'warning.main', borderRadius: 2, p: 1.5 }}>
                        <Typography sx={{ fontWeight: 600 }}>{label(c.rowId)}</Typography>
                        <Typography variant="body2" color="warning.main">⚠ {t('collaboration.conflicts.mustChange')}</Typography>
                    </Box>
                </Box>
                <Typography variant="body2" color="text.secondary">{t('collaboration.conflicts.rule')}</Typography>
                {!canWrite ? (
                    <Alert severity="info">{t('collaboration.conflicts.readOnly')}</Alert>
                ) : (
                    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'flex-start' }}>
                        {c.column === 'number' && (
                            <Button variant="contained" onClick={() => save(c, String(nextFreeNumber()))}>
                                {t('collaboration.conflicts.nextFree', { number: nextFreeNumber() })}
                            </Button>
                        )}
                        <TextField
                            id={`conflict-${k}`}
                            size="small"
                            label={c.column === 'number' ? t('collaboration.conflicts.chooseOwn')
                                : c.column === 'email' ? t('collaboration.conflicts.changeEmail') : t('collaboration.conflicts.rename')}
                            value={input[k] ?? ''}
                            onChange={(e) => setInput({ ...input, [k]: e.target.value })}
                            inputProps={c.column === 'number' ? { inputMode: 'numeric' } : undefined}
                        />
                        <Button variant="outlined" disabled={!input[k]?.trim()} onClick={() => save(c, input[k].trim())}>
                            {t('collaboration.conflicts.apply')}
                        </Button>
                    </Box>
                )}
            </Paper>
        );
    };

    return (
        <Box sx={{ p: { xs: 2, sm: 4 } }}>
            <PageHeader title={t('collaboration.conflicts.title')} icon={<ReportProblemOutlinedIcon fontSize="small" />} />
            <Stack spacing={2}>
                {error && <Alert severity="error" onClose={() => setError('')}>{error}</Alert>}
                {conflicts === null ? <CircularProgress size={24} />
                    : conflicts.length === 0 ? <Paper sx={{ p: 3 }}><Typography>{t('collaboration.conflicts.none')}</Typography></Paper>
                    : conflicts.map(card)}
            </Stack>
        </Box>
    );
}
