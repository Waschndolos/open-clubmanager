import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Alert,
    Box,
    Button,
    Chip,
    LinearProgress,
    ListSubheader,
    MenuItem,
    Paper,
    Tab,
    Tabs,
    TextField,
    Typography,
} from '@mui/material';
import { Download, FactCheckOutlined } from '@mui/icons-material';
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import PageHeader from '../../components/common/PageHeader';
import ChartCard, { useChartStyle } from '../../components/dashboard/ChartCard';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';
import { fetchGroups } from '../../api/groups';
import { fetchSections } from '../../api/sections';
import { canEditData } from '../../api/types';
import { addDays, CATEGORY_COLOR, NamedRef, ymd } from '../../api/events';
import {
    AttendanceFilter,
    AttendanceOccurrence,
    AttendanceStats,
    downloadAttendance,
    fetchAttendanceOccurrences,
    fetchAttendanceStats,
} from '../../api/attendance';
import { useAttendanceSync } from './useAttendanceSync';
import AttendanceSheetView from './AttendanceSheetView';

export default function Attendance() {
    const { t } = useTranslation();
    const { appRole } = useAuth();
    const area = usePermission('membersCore');
    const canEdit = area === 'write' && (canEditData(appRole) || appRole === 'TRAINER');
    const [params, setParams] = useSearchParams();
    const [tab, setTab] = useState<'record' | 'stats'>('record');
    const pending = useAttendanceSync();

    const eventId = params.get('event');
    const date = params.get('date');
    const open = (o: { eventId: string; occurrenceDate: string }) => setParams({ event: o.eventId, date: o.occurrenceDate });

    return (
        <Box p={3}>
            <PageHeader title={t('attendance.title')} subtitle={t('attendance.subtitle')} icon={<FactCheckOutlined fontSize="small" />} />
            {eventId && date ? (
                <AttendanceSheetView eventId={eventId} date={date} canEdit={canEdit} onBack={() => setParams({})} />
            ) : (
                <>
                    {pending > 0 && <Alert severity="warning" sx={{ mb: 2 }}>{t('attendance.pending', { count: pending })}</Alert>}
                    <Tabs value={tab} onChange={(_e, v) => setTab(v)} sx={{ mb: 2 }}>
                        <Tab value="record" label={t('attendance.tabs.record')} />
                        <Tab value="stats" label={t('attendance.tabs.stats')} />
                    </Tabs>
                    {tab === 'record' ? <OccurrencePicker onOpen={open} /> : <Statistics />}
                </>
            )}
        </Box>
    );
}

function useAudienceOptions() {
    const [groups, setGroups] = useState<NamedRef[]>([]);
    const [sections, setSections] = useState<NamedRef[]>([]);
    useEffect(() => {
        fetchGroups().then(setGroups).catch(() => undefined);
        fetchSections().then(setSections).catch(() => undefined);
    }, []);
    return { groups, sections };
}

function AudienceSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
    const { t } = useTranslation();
    const { groups, sections } = useAudienceOptions();
    return (
        <TextField select size="small" label={t('attendance.filter')} value={value} onChange={(e) => onChange(e.target.value)} sx={{ minWidth: 200 }}>
            <MenuItem value="">{t('attendance.allGroups')}</MenuItem>
            {groups.length > 0 && <ListSubheader>{t('events.fields.groups')}</ListSubheader>}
            {groups.map((g) => <MenuItem key={g.id} value={`group:${g.id}`}>{g.name}</MenuItem>)}
            {sections.length > 0 && <ListSubheader>{t('events.fields.sections')}</ListSubheader>}
            {sections.map((s) => <MenuItem key={s.id} value={`section:${s.id}`}>{s.name}</MenuItem>)}
        </TextField>
    );
}

function toFilter(key: string): AttendanceFilter {
    const [kind, id] = key.split(':');
    if (!id) return {};
    return kind === 'group' ? { groupId: id } : { sectionId: id };
}

/** Recent and today's occurrences, newest first, to pick one for recording. */
function OccurrencePicker({ onOpen }: { onOpen: (o: AttendanceOccurrence) => void }) {
    const { t, i18n } = useTranslation();
    const { addNotification } = useNotification();
    const [days, setDays] = useState(14);
    const [filterKey, setFilterKey] = useState('');
    const [list, setList] = useState<AttendanceOccurrence[] | null>(null);

    useEffect(() => {
        const today = new Date();
        fetchAttendanceOccurrences(ymd(addDays(today, -days)), ymd(addDays(today, 1)), toFilter(filterKey))
            .then((l) => setList([...l].reverse()))
            .catch(() => {
                setList([]);
                addNotification(t('attendance.loadFailed'));
            });
    }, [days, filterKey, addNotification, t]);

    return (
        <Box>
            <Box display="flex" gap={2} mb={2} flexWrap="wrap">
                <AudienceSelect value={filterKey} onChange={setFilterKey} />
                <TextField select size="small" label={t('attendance.period')} value={days} onChange={(e) => setDays(Number(e.target.value))} sx={{ minWidth: 180 }}>
                    {[7, 14, 30, 90].map((d) => <MenuItem key={d} value={d}>{t('attendance.lastDays', { count: d })}</MenuItem>)}
                </TextField>
            </Box>
            {list === null && <LinearProgress />}
            {list?.length === 0 && <Typography color="text.secondary">{t('attendance.noOccurrences')}</Typography>}
            <Box display="grid" gap={1.5} gridTemplateColumns="repeat(auto-fill, minmax(280px, 1fr))">
                {list?.map((o) => {
                    const start = new Date(o.start);
                    const recorded = o.present + o.excused + o.unexcused;
                    return (
                        <Paper key={`${o.eventId}-${o.occurrenceDate}`} variant="outlined" onClick={() => onOpen(o)}
                               role="button" tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter') onOpen(o); }}
                               sx={{ p: 2, cursor: 'pointer', '&:hover': { bgcolor: 'action.hover' } }}>
                            <Typography variant="caption" color="text.secondary">
                                {start.toLocaleDateString(i18n.language, { weekday: 'short', day: 'numeric', month: 'short' })}
                                {!o.allDay && ` · ${start.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })}`}
                            </Typography>
                            <Typography fontWeight={600} noWrap>{o.title}</Typography>
                            <Box display="flex" gap={0.5} mt={1} flexWrap="wrap">
                                <Chip size="small" color={CATEGORY_COLOR[o.category]} label={t(`events.category.${o.category}`)} />
                                {recorded > 0
                                    ? <Chip size="small" variant="outlined" color="success" label={t('attendance.recorded', { present: o.present, count: recorded })} />
                                    : <Chip size="small" variant="outlined" label={t('attendance.notRecorded')} />}
                            </Box>
                        </Paper>
                    );
                })}
            </Box>
        </Box>
    );
}

/** Attendance per member and over time, with CSV/PDF export. */
function Statistics() {
    const { t, i18n } = useTranslation();
    const { addNotification } = useNotification();
    const s = useChartStyle();
    const [from, setFrom] = useState(() => `${new Date().getFullYear()}-01-01`);
    const [to, setTo] = useState(() => ymd(new Date()));
    const [filterKey, setFilterKey] = useState('');
    const [stats, setStats] = useState<AttendanceStats | null>(null);
    const valid = from !== '' && to !== '' && from <= to;

    const reload = useCallback(() => {
        if (!valid) return;
        fetchAttendanceStats(from, to, toFilter(filterKey)).then(setStats).catch(() => addNotification(t('attendance.loadFailed')));
    }, [from, to, filterKey, valid, addNotification, t]);
    useEffect(reload, [reload]);

    const history = useMemo(() => (stats?.occurrences ?? []).map((o) => {
        const total = o.present + o.excused + o.unexcused;
        return {
            label: new Date(o.start).toLocaleDateString(i18n.language, { day: '2-digit', month: '2-digit' }),
            title: o.title,
            present: o.present,
            rate: total ? Math.round((o.present / total) * 100) : 0,
        };
    }), [stats, i18n.language]);

    const exportAs = (format: 'csv' | 'pdf') =>
        downloadAttendance(from, to, format, toFilter(filterKey)).catch(() => addNotification(t('attendance.exportFailed')));

    return (
        <Box display="flex" flexDirection="column" gap={2}>
            <Box display="flex" gap={2} flexWrap="wrap" alignItems="center">
                <TextField size="small" type="date" label={t('attendance.from')} value={from} onChange={(e) => setFrom(e.target.value)}
                           error={!valid} slotProps={{ inputLabel: { shrink: true } }} />
                <TextField size="small" type="date" label={t('attendance.to')} value={to} onChange={(e) => setTo(e.target.value)}
                           error={!valid} slotProps={{ inputLabel: { shrink: true } }} />
                <AudienceSelect value={filterKey} onChange={setFilterKey} />
                <Box flexGrow={1} />
                <Button variant="outlined" startIcon={<Download />} disabled={!valid} onClick={() => void exportAs('csv')}>CSV</Button>
                <Button variant="outlined" startIcon={<Download />} disabled={!valid} onClick={() => void exportAs('pdf')}>PDF</Button>
            </Box>

            <ChartCard title={t('attendance.history')} empty={history.length === 0} height={220}>
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={history} margin={{ top: 5, right: 8, left: -16, bottom: 0 }}>
                        <CartesianGrid stroke={s.grid} strokeDasharray="3 3" vertical={false} />
                        <XAxis dataKey="label" tick={s.tick} tickLine={false} axisLine={false} />
                        <YAxis tick={s.tick} tickLine={false} axisLine={false} unit="%" domain={[0, 100]} />
                        <Tooltip {...s.tooltip} formatter={(v) => [`${v} %`, t('attendance.rate')]}
                                 labelFormatter={(_l, p) => (p?.[0]?.payload ? `${p[0].payload.label} ${p[0].payload.title}` : '')} />
                        <Line type="monotone" dataKey="rate" stroke={s.positive} strokeWidth={2} dot={{ r: 3 }} />
                    </LineChart>
                </ResponsiveContainer>
            </ChartCard>

            <Paper variant="outlined">
                <Box sx={{ display: 'grid', gridTemplateColumns: 'minmax(160px, 2fr) repeat(3, minmax(70px, 1fr)) minmax(120px, 1.5fr)',
                           px: 2, py: 1, fontWeight: 600, fontSize: '0.85rem', color: 'text.secondary', borderBottom: 1, borderColor: 'divider' }}>
                    <span>{t('attendance.member')}</span>
                    <span>{t('attendance.status.present')}</span>
                    <span>{t('attendance.status.excused')}</span>
                    <span>{t('attendance.status.unexcused')}</span>
                    <span>{t('attendance.rate')}</span>
                </Box>
                {stats?.members.length === 0 && <Typography color="text.secondary" sx={{ p: 2 }}>{t('attendance.noRecords')}</Typography>}
                {stats?.members.map((m) => (
                    <Box key={m.member.id} sx={{ display: 'grid', gridTemplateColumns: 'minmax(160px, 2fr) repeat(3, minmax(70px, 1fr)) minmax(120px, 1.5fr)',
                                                 alignItems: 'center', px: 2, py: 1, '&:not(:last-of-type)': { borderBottom: 1, borderColor: 'divider' } }}>
                        <Typography noWrap>{m.member.lastName}, {m.member.firstName}</Typography>
                        <span>{m.present}</span>
                        <span>{m.excused}</span>
                        <span>{m.unexcused}</span>
                        <Box display="flex" alignItems="center" gap={1}>
                            <LinearProgress variant="determinate" value={m.rate * 100} sx={{ flexGrow: 1, height: 6, borderRadius: 3 }}
                                            color={m.rate >= 0.75 ? 'success' : m.rate >= 0.5 ? 'warning' : 'error'} />
                            <Typography variant="body2" sx={{ width: 40, textAlign: 'right' }}>{Math.round(m.rate * 100)} %</Typography>
                        </Box>
                    </Box>
                ))}
            </Paper>
        </Box>
    );
}
