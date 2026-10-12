import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    Paper,
    Tab,
    Tabs,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { CelebrationOutlined, Delete, Download, EditCalendar, MilitaryTech } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';
import {
    createHonor,
    DEFAULT_AGES,
    DEFAULT_YEARS,
    deleteHonor,
    downloadJubileesCsv,
    fetchHonors,
    fetchJubilees,
    fetchMembershipAdjust,
    Honor,
    isoDate,
    Jubilee,
    JubileeQuery,
    setMembershipAdjust,
} from '../../api/honors';

const STORAGE_KEY = 'jubilee-stages';

function parseList(text: string): number[] {
    return text.split(',').map((s) => parseInt(s.trim(), 10)).filter((n) => Number.isFinite(n) && n > 0 && n <= 200);
}

function loadStages(): { years: string; ages: string } {
    try {
        const saved = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? 'null');
        if (typeof saved?.years === 'string' && typeof saved?.ages === 'string') return saved;
    } catch {
        // unavailable or malformed storage: use the defaults
    }
    return { years: DEFAULT_YEARS.join(', '), ages: DEFAULT_AGES.join(', ') };
}

function defaultRange() {
    const from = new Date();
    const to = new Date();
    to.setDate(to.getDate() + 90);
    return { from: isoDate(from), to: isoDate(to) };
}

export default function Honors() {
    const { t, i18n } = useTranslation();
    const { appRole } = useAuth();
    const { addNotification } = useNotification();
    const area = usePermission('membersCore');
    const canEdit = (appRole === 'ADMIN' || appRole === 'SECRETARY') && area === 'write';

    const [tab, setTab] = useState(0);
    const [range, setRange] = useState(defaultRange);
    const [stages, setStages] = useState(loadStages);
    const [jubilees, setJubilees] = useState<Jubilee[]>([]);
    const [honors, setHonors] = useState<Honor[]>([]);
    const [honorFor, setHonorFor] = useState<Jubilee | null>(null);
    const [adjustFor, setAdjustFor] = useState<Jubilee | null>(null);

    const query = useMemo<JubileeQuery>(() => ({
        from: range.from, to: range.to, years: parseList(stages.years), ages: parseList(stages.ages),
    }), [range, stages]);

    const reload = useCallback(async () => {
        try {
            const [j, h] = await Promise.all([fetchJubilees(query), fetchHonors()]);
            setJubilees(j);
            setHonors(h);
        } catch {
            addNotification(t('honors.loadFailed'));
        }
    }, [query, addNotification, t]);

    useEffect(() => {
        if (range.from && range.to && range.from <= range.to) void reload();
    }, [range, reload]);

    const changeStages = (patch: Partial<typeof stages>) => {
        const next = { ...stages, ...patch };
        setStages(next);
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
        } catch {
            // not persisted; the choice still applies to this session
        }
    };

    const fmtDate = (iso: string) => new Date(`${iso}T00:00:00`).toLocaleDateString(i18n.language);
    const memberName = (m: { number: number; firstName: string; lastName: string }) => `${m.firstName} ${m.lastName} (${m.number})`;

    const jubileeCols: GridColDef<Jubilee>[] = [
        { field: 'date', headerName: t('honors.columns.date'), width: 130, valueFormatter: (v: string) => fmtDate(v) },
        { field: 'member', headerName: t('honors.columns.member'), flex: 1, minWidth: 200, valueGetter: (_v, r) => memberName(r.member) },
        { field: 'kind', headerName: t('honors.columns.occasion'), flex: 1, minWidth: 180, valueGetter: (_v, r) => t(`honors.kind.${r.kind}`, { count: r.milestone }) },
        ...(canEdit ? [{
            field: 'actions', headerName: '', width: 110, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <>
                    <Tooltip title={t('honors.record')}>
                        <IconButton aria-label={t('honors.record')} onClick={() => setHonorFor(row)}><MilitaryTech /></IconButton>
                    </Tooltip>
                    {row.kind === 'membership' && (
                        <Tooltip title={t('honors.adjust.title')}>
                            <IconButton aria-label={t('honors.adjust.title')} onClick={() => setAdjustFor(row)}><EditCalendar /></IconButton>
                        </Tooltip>
                    )}
                </>
            ),
        } as GridColDef<Jubilee>] : []),
    ];

    const honorCols: GridColDef<Honor>[] = [
        { field: 'date', headerName: t('honors.columns.date'), width: 130, valueFormatter: (v: string) => fmtDate(v) },
        { field: 'member', headerName: t('honors.columns.member'), flex: 1, minWidth: 200, valueGetter: (_v, r) => memberName(r.member) },
        { field: 'title', headerName: t('honors.columns.honor'), flex: 1, minWidth: 180 },
        { field: 'kind', headerName: t('honors.columns.occasion'), flex: 1, minWidth: 160, valueGetter: (_v, r) => r.kind === 'other' ? t('honors.kind.other') : t(`honors.kind.${r.kind}`, { count: r.milestone }) },
        ...(canEdit ? [{
            field: 'actions', headerName: '', width: 70, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <IconButton aria-label={t('honors.delete')} onClick={async () => {
                    if (!window.confirm(t('honors.confirmDelete'))) return;
                    try {
                        await deleteHonor(row.id);
                        await reload();
                    } catch {
                        addNotification(t('honors.saveFailed'));
                    }
                }}><Delete /></IconButton>
            ),
        } as GridColDef<Honor>] : []),
    ];

    const exportCsv = async () => {
        try {
            await downloadJubileesCsv(query);
        } catch {
            addNotification(t('honors.exportFailed'));
        }
    };

    return (
        <Box p={3}>
            <PageHeader
                title={t('honors.title')}
                subtitle={t('honors.subtitle')}
                icon={<CelebrationOutlined fontSize="small" />}
                actions={
                    <Button variant="outlined" startIcon={<Download />} onClick={exportCsv} disabled={jubilees.length === 0}>
                        {t('honors.export')}
                    </Button>
                }
            />

            <Paper variant="outlined" sx={{ p: 2, mb: 2, display: 'flex', flexWrap: 'wrap', gap: 2 }}>
                <TextField type="date" size="small" label={t('honors.from')} value={range.from}
                           onChange={(e) => setRange({ ...range, from: e.target.value })} slotProps={{ inputLabel: { shrink: true } }} />
                <TextField type="date" size="small" label={t('honors.to')} value={range.to}
                           onChange={(e) => setRange({ ...range, to: e.target.value })} slotProps={{ inputLabel: { shrink: true } }} />
                <TextField size="small" label={t('honors.stages.years')} value={stages.years} helperText={t('honors.stages.hint')}
                           onChange={(e) => changeStages({ years: e.target.value })} />
                <TextField size="small" label={t('honors.stages.ages')} value={stages.ages}
                           onChange={(e) => changeStages({ ages: e.target.value })} />
            </Paper>

            <Tabs value={tab} onChange={(_e, v) => setTab(v)} sx={{ mb: 2 }}>
                <Tab label={`${t('honors.tabs.upcoming')} (${jubilees.length})`} />
                <Tab label={`${t('honors.tabs.given')} (${honors.length})`} />
            </Tabs>

            {tab === 0 ? (
                <DataGrid rows={jubilees} columns={jubileeCols} autoHeight disableRowSelectionOnClick
                          getRowId={(r) => `${r.member.id}-${r.kind}-${r.milestone}`}
                          localeText={{ noRowsLabel: t('honors.empty') }} />
            ) : (
                <DataGrid rows={honors} columns={honorCols} autoHeight disableRowSelectionOnClick
                          localeText={{ noRowsLabel: t('honors.emptyGiven') }} />
            )}

            {honorFor && (
                <HonorDialog jubilee={honorFor} onClose={() => setHonorFor(null)} onSaved={async () => {
                    setHonorFor(null);
                    await reload();
                }} />
            )}
            {adjustFor && (
                <AdjustDialog jubilee={adjustFor} onClose={() => setAdjustFor(null)} onSaved={async () => {
                    setAdjustFor(null);
                    await reload();
                }} />
            )}
        </Box>
    );
}

function HonorDialog({ jubilee, onClose, onSaved }: { jubilee: Jubilee; onClose: () => void; onSaved: () => Promise<void> }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [title, setTitle] = useState('');
    const [date, setDate] = useState(isoDate(new Date()));
    const [note, setNote] = useState('');

    const save = async () => {
        try {
            await createHonor(jubilee.member.id, {
                kind: jubilee.kind, milestone: jubilee.milestone, title: title.trim(), date, note: note.trim() || undefined,
            });
            await onSaved();
        } catch {
            addNotification(t('honors.saveFailed'));
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{t('honors.record')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <Typography>
                        {jubilee.member.firstName} {jubilee.member.lastName} – {t(`honors.kind.${jubilee.kind}`, { count: jubilee.milestone })}
                    </Typography>
                    <TextField autoFocus required label={t('honors.columns.honor')} placeholder={t('honors.titleHint')}
                               value={title} onChange={(e) => setTitle(e.target.value)} />
                    <TextField type="date" label={t('honors.columns.date')} value={date}
                               onChange={(e) => setDate(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                    <TextField multiline minRows={2} label={t('honors.note')} value={note} onChange={(e) => setNote(e.target.value)} />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={save} disabled={!title.trim() || !date}>{t('buttons.save')}</Button>
            </DialogActions>
        </Dialog>
    );
}

function AdjustDialog({ jubilee, onClose, onSaved }: { jubilee: Jubilee; onClose: () => void; onSaved: () => Promise<void> }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [years, setYears] = useState('0');

    useEffect(() => {
        fetchMembershipAdjust(jubilee.member.id).then((y) => setYears(String(y))).catch(() => undefined);
    }, [jubilee.member.id]);

    const value = parseInt(years, 10);
    const save = async () => {
        try {
            await setMembershipAdjust(jubilee.member.id, value);
            await onSaved();
        } catch {
            addNotification(t('honors.saveFailed'));
        }
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="xs">
            <DialogTitle>{t('honors.adjust.title')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <Typography>{jubilee.member.firstName} {jubilee.member.lastName}</Typography>
                    <TextField type="number" label={t('honors.adjust.years')} helperText={t('honors.adjust.hint')}
                               value={years} onChange={(e) => setYears(e.target.value)} slotProps={{ htmlInput: { min: -100, max: 100 } }} />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={save} disabled={!Number.isInteger(value) || value < -100 || value > 100}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
