import { ReactElement, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
    Alert,
    Autocomplete,
    Box,
    Button,
    Chip,
    CircularProgress,
    IconButton,
    InputAdornment,
    Paper,
    TextField,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
    Typography,
} from '@mui/material';
import { ArrowBack, Cancel, CheckCircle, Clear, CloudOff, DoneAll, HelpOutline, Search } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { useNotification } from '../../context/NotificationContext';
import { fetchMembers } from '../../api/members';
import { Member } from '../../api/types';
import {
    AttendanceEntry,
    AttendanceMember,
    AttendanceSheet,
    AttendanceStatus,
    fetchAttendanceSheet,
    queuedEntries,
    saveAttendance,
} from '../../api/attendance';

const STATUS_ICON: Record<AttendanceStatus, ReactElement> = {
    present: <CheckCircle fontSize="small" />,
    excused: <HelpOutline fontSize="small" />,
    unexcused: <Cancel fontSize="small" />,
};
const STATUS_COLOR: Record<AttendanceStatus, 'success' | 'warning' | 'error'> = {
    present: 'success',
    excused: 'warning',
    unexcused: 'error',
};

type Props = {
    eventId: string;
    date: string;
    canEdit: boolean;
    onBack: () => void;
};

/** Quick attendance capture for one occurrence: tap a state per member. */
export default function AttendanceSheetView({ eventId, date, canEdit, onBack }: Props) {
    const { t, i18n } = useTranslation();
    const { addNotification } = useNotification();
    const [sheet, setSheet] = useState<AttendanceSheet | null>(null);
    const [members, setMembers] = useState<AttendanceMember[]>([]);
    const [search, setSearch] = useState('');
    // Changes of this occurrence still waiting in the offline queue.
    const [queued, setQueued] = useState(() => queuedEntries(eventId, date).length > 0);
    const [allMembers, setAllMembers] = useState<Member[]>([]);
    // Saves run one after another so quick taps cannot overtake each other.
    const chain = useRef<Promise<unknown>>(Promise.resolve());

    const load = useCallback(async () => {
        try {
            const s = await fetchAttendanceSheet(eventId, date);
            // Apply changes still waiting in the offline queue.
            const queued = new Map(queuedEntries(eventId, date).map((e) => [e.memberId, e.status]));
            setSheet(s);
            setMembers(s.members.map((m) => queued.has(m.id) ? { ...m, status: queued.get(m.id) ?? undefined } : m));
        } catch {
            addNotification(t('attendance.loadFailed'));
        }
    }, [eventId, date, addNotification, t]);
    useEffect(() => { void load(); }, [load]);

    useEffect(() => {
        if (canEdit) fetchMembers().then(setAllMembers).catch(() => undefined);
    }, [canEdit]);

    useEffect(() => {
        const update = () => setQueued(queuedEntries(eventId, date).length > 0);
        update();
        window.addEventListener('attendance-queue', update);
        return () => window.removeEventListener('attendance-queue', update);
    }, [eventId, date]);

    const save = (entries: AttendanceEntry[]) => {
        chain.current = chain.current.then(async () => {
            try {
                await saveAttendance(eventId, date, entries);
            } catch {
                addNotification(t('attendance.saveFailed'));
                await load();
            }
        });
    };

    const setStatus = (memberIds: string[], status: AttendanceStatus | null) => {
        const ids = new Set(memberIds);
        setMembers((prev) => prev.map((m) => ids.has(m.id) ? { ...m, status: status ?? undefined } : m));
        save(memberIds.map((memberId) => ({ memberId, status })));
    };

    const counts = useMemo(() => {
        const c = { present: 0, excused: 0, unexcused: 0, open: 0 };
        for (const m of members) {
            if (m.status) c[m.status]++;
            else c.open++;
        }
        return c;
    }, [members]);

    const visible = members.filter((m) =>
        `${m.firstName} ${m.lastName} ${m.number}`.toLowerCase().includes(search.trim().toLowerCase()));
    const onSheet = new Set(members.map((m) => m.id));

    if (!sheet) return <CircularProgress />;
    const o = sheet.occurrence;
    const start = new Date(o.start);

    return (
        <Box>
            <Box display="flex" alignItems="center" gap={1} mb={2} flexWrap="wrap">
                <IconButton aria-label={t('attendance.back')} onClick={onBack}><ArrowBack /></IconButton>
                <Box flexGrow={1} minWidth={0}>
                    <Typography variant="h6" component="h2" noWrap>{o.title}</Typography>
                    <Typography variant="body2" color="text.secondary">
                        {start.toLocaleDateString(i18n.language, { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })}
                        {!o.allDay && `, ${start.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })}`}
                    </Typography>
                </Box>
                {/* One line of fixed height that scrolls instead of wrapping, so
                    the rows below never move while someone is tapping. */}
                <Box display="flex" gap={0.5} flexWrap="nowrap" alignItems="center"
                     sx={{ flexBasis: '100%', height: 32, overflowX: 'auto', '& .MuiChip-root': { flexShrink: 0 } }}>
                    <Chip size="small" color="success" label={`${counts.present} ${t('attendance.status.present')}`} />
                    <Chip size="small" color="warning" label={`${counts.excused} ${t('attendance.status.excused')}`} />
                    <Chip size="small" color="error" label={`${counts.unexcused} ${t('attendance.status.unexcused')}`} />
                    {counts.open > 0 && <Chip size="small" variant="outlined" label={`${counts.open} ${t('attendance.open')}`} />}
                    {/* A chip in this row, not a banner above the list, so rows do not move under the finger. */}
                    {queued && (
                        <Tooltip title={t('attendance.offlineSaved')}>
                            <Chip size="small" color="info" icon={<CloudOff />} label={t('attendance.offline')} />
                        </Tooltip>
                    )}
                </Box>
            </Box>

            {!canEdit && <Alert severity="info" sx={{ mb: 2 }}>{t('attendance.readOnly')}</Alert>}

            <Box display="flex" gap={1} mb={2} flexWrap="wrap" alignItems="center">
                <TextField size="small" placeholder={t('attendance.search')} value={search} onChange={(e) => setSearch(e.target.value)}
                           sx={{ flex: '1 1 200px' }}
                           slotProps={{ input: {
                               startAdornment: <InputAdornment position="start"><Search fontSize="small" /></InputAdornment>,
                               endAdornment: search && <IconButton size="small" aria-label={t('attendance.clearSearch')} onClick={() => setSearch('')}><Clear fontSize="small" /></IconButton>,
                           } }} />
                {canEdit && counts.open > 0 && (
                    <Button variant="contained" startIcon={<DoneAll />}
                            onClick={() => setStatus(members.filter((m) => !m.status).map((m) => m.id), 'present')}>
                        {t('attendance.allPresent', { count: counts.open })}
                    </Button>
                )}
            </Box>

            {members.length === 0 && <Typography color="text.secondary">{t('attendance.noMembers')}</Typography>}
            <Paper variant="outlined">
                {visible.map((m) => (
                    <Box key={m.id} sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 1.5, py: 1, flexWrap: 'wrap',
                                          '&:not(:last-of-type)': { borderBottom: 1, borderColor: 'divider' } }}>
                        <Box flexGrow={1} minWidth={140}>
                            <Typography fontWeight={600}>{m.lastName}, {m.firstName}</Typography>
                            <Typography variant="caption" color="text.secondary">
                                #{m.number}{!m.inAudience && ` · ${t('attendance.guest')}`}
                            </Typography>
                        </Box>
                        <ToggleButtonGroup exclusive size="medium" value={m.status ?? null} disabled={!canEdit}
                                           aria-label={`${m.firstName} ${m.lastName}`}
                                           onChange={(_e, v: AttendanceStatus | null) => setStatus([m.id], v)}>
                            {(['present', 'excused', 'unexcused'] as AttendanceStatus[]).map((s) => (
                                <ToggleButton key={s} value={s} color={STATUS_COLOR[s]} aria-label={t(`attendance.status.${s}`)}
                                              sx={{ px: { xs: 1.5, sm: 2 }, py: 1, gap: 0.75 }}>
                                    <Tooltip title={t(`attendance.status.${s}`)}>{STATUS_ICON[s]}</Tooltip>
                                    <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' } }}>{t(`attendance.status.${s}`)}</Box>
                                </ToggleButton>
                            ))}
                        </ToggleButtonGroup>
                    </Box>
                ))}
            </Paper>

            {canEdit && (
                <Autocomplete sx={{ mt: 2, maxWidth: 420 }} size="small" value={null} blurOnSelect clearOnBlur
                              options={allMembers.filter((m) => !onSheet.has(m.id))}
                              getOptionLabel={(m) => `${m.firstName} ${m.lastName} (${m.number})`}
                              onChange={(_e, m) => {
                                  if (!m) return;
                                  setMembers((prev) => [...prev, { id: m.id, number: m.number ?? 0, firstName: m.firstName ?? '',
                                      lastName: m.lastName ?? '', inAudience: false }]);
                              }}
                              renderInput={(p) => <TextField {...p} label={t('attendance.addGuest')} />} />
            )}
        </Box>
    );
}
