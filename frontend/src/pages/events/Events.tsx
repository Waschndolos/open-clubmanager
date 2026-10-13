import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    ListSubheader,
    MenuItem,
    Paper,
    TextField,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
    Typography,
} from '@mui/material';
import { Add, ChevronLeft, ChevronRight, Download, EventOutlined, PlaceOutlined, Repeat } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import PageHeader from '../../components/common/PageHeader';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';
import { fetchGroups } from '../../api/groups';
import { fetchRoles } from '../../api/roles';
import { fetchSections } from '../../api/sections';
import { canEditData } from '../../api/types';
import {
    addDays,
    AudienceFilter,
    CATEGORY_COLOR,
    ClubEvent,
    coversDay,
    createEvent,
    deleteEvent,
    downloadCalendarIcs,
    downloadEventIcs,
    EventInput,
    EventOccurrence,
    excludeOccurrence,
    fetchEvent,
    fetchOccurrences,
    NamedRef,
    startOfWeek,
    updateEvent,
    ymd,
} from '../../api/events';
import EventDialog from './EventDialog';

type View = 'month' | 'week' | 'list';
const LIST_DAYS = 90;

export default function Events() {
    const { t, i18n } = useTranslation();
    const { appRole } = useAuth();
    const area = usePermission('membersCore');
    const canEdit = area === 'write' && canEditData(appRole);
    const { addNotification } = useNotification();
    const [params, setParams] = useSearchParams();

    const [view, setView] = useState<View>('month');
    const [anchor, setAnchor] = useState(() => new Date());
    const [filterKey, setFilterKey] = useState('');
    const [occurrences, setOccurrences] = useState<EventOccurrence[]>([]);
    const [groups, setGroups] = useState<NamedRef[]>([]);
    const [sections, setSections] = useState<NamedRef[]>([]);
    const [roles, setRoles] = useState<NamedRef[]>([]);
    const [selected, setSelected] = useState<EventOccurrence | null>(null);
    const [editing, setEditing] = useState<{ event?: ClubEvent; date?: Date } | null>(null);

    useEffect(() => {
        fetchGroups().then(setGroups).catch(() => undefined);
        fetchSections().then(setSections).catch(() => undefined);
        fetchRoles().then(setRoles).catch(() => undefined);
    }, []);

    // Deep link from the command palette or dashboard: ?new=1 opens the dialog.
    useEffect(() => {
        if (params.get('new') === '1') {
            setEditing({});
            setParams({}, { replace: true });
        }
    }, [params, setParams]);

    const filter: AudienceFilter = useMemo(() => {
        const [kind, id] = filterKey.split(':');
        if (!id) return {};
        return kind === 'group' ? { groupId: id } : kind === 'section' ? { sectionId: id } : { roleId: id };
    }, [filterKey]);

    // Visible days of the current view.
    const range = useMemo(() => {
        if (view === 'week') {
            const from = startOfWeek(anchor);
            return { from, to: addDays(from, 6) };
        }
        if (view === 'list') {
            const from = new Date(anchor.getFullYear(), anchor.getMonth(), anchor.getDate());
            return { from, to: addDays(from, LIST_DAYS - 1) };
        }
        const from = startOfWeek(new Date(anchor.getFullYear(), anchor.getMonth(), 1));
        return { from, to: addDays(from, 41) };
    }, [view, anchor]);

    const reload = useCallback(async () => {
        try {
            setOccurrences(await fetchOccurrences(ymd(range.from), ymd(range.to), filter));
        } catch {
            addNotification(t('events.loadFailed'));
        }
    }, [range, filter, addNotification, t]);
    useEffect(() => { void reload(); }, [reload]);

    const run = async (fn: () => Promise<unknown>, success?: string) => {
        try {
            await fn();
            if (success) addNotification(success);
        } catch {
            addNotification(t('events.saveFailed'));
        }
        await reload();
    };

    const move = (dir: number) => {
        const d = new Date(anchor);
        if (view === 'month') d.setMonth(d.getMonth() + dir, 1);
        else if (view === 'week') d.setDate(d.getDate() + 7 * dir);
        else d.setDate(d.getDate() + LIST_DAYS * dir);
        setAnchor(d);
    };

    const title = view === 'month'
        ? anchor.toLocaleDateString(i18n.language, { month: 'long', year: 'numeric' })
        : `${range.from.toLocaleDateString(i18n.language, { day: 'numeric', month: 'short' })} – ${range.to.toLocaleDateString(i18n.language, { day: 'numeric', month: 'short', year: 'numeric' })}`;

    const time = (o: EventOccurrence) => o.allDay ? t('events.allDay')
        : new Date(o.start).toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' });

    const days = useMemo(() => {
        const out: Date[] = [];
        for (let d = new Date(range.from); d <= range.to; d = addDays(d, 1)) out.push(d);
        return out;
    }, [range]);
    const weekdays = days.slice(0, 7).map((d) => d.toLocaleDateString(i18n.language, { weekday: 'short' }));
    const today = ymd(new Date());

    const chip = (o: EventOccurrence, dense = false) => (
        <Chip key={`${o.eventId}-${o.occurrenceDate}`} size="small" color={CATEGORY_COLOR[o.category]} clickable
              variant={o.category === 'other' ? 'outlined' : 'filled'}
              label={dense && !o.allDay ? `${time(o)} ${o.title}` : o.title}
              onClick={() => setSelected(o)}
              sx={{ maxWidth: '100%', justifyContent: 'flex-start', '& .MuiChip-label': { overflow: 'hidden', textOverflow: 'ellipsis' } }} />
    );

    return (
        <Box p={3}>
            <PageHeader
                title={t('events.title')}
                subtitle={t('events.subtitle')}
                icon={<EventOutlined fontSize="small" />}
                actions={<>
                    <Button variant="outlined" startIcon={<Download />}
                            onClick={() => void run(() => downloadCalendarIcs(filter))}>{t('events.exportCalendar')}</Button>
                    {canEdit && <Button variant="contained" startIcon={<Add />} onClick={() => setEditing({})}>{t('events.add')}</Button>}
                </>}
            />

            <Box display="flex" flexWrap="wrap" alignItems="center" gap={2} mb={2}>
                <Box display="flex" alignItems="center">
                    <IconButton aria-label={t('events.previous')} onClick={() => move(-1)}><ChevronLeft /></IconButton>
                    <Button size="small" onClick={() => setAnchor(new Date())}>{t('events.today')}</Button>
                    <IconButton aria-label={t('events.next')} onClick={() => move(1)}><ChevronRight /></IconButton>
                    <Typography variant="h6" component="h2" sx={{ ml: 1, textTransform: 'capitalize' }}>{title}</Typography>
                </Box>
                <Box flexGrow={1} />
                <TextField select size="small" label={t('events.audience')} value={filterKey} sx={{ minWidth: 200 }}
                           onChange={(e) => setFilterKey(e.target.value)}>
                    <MenuItem value="">{t('events.allAudiences')}</MenuItem>
                    {groups.length > 0 && <ListSubheader>{t('events.fields.groups')}</ListSubheader>}
                    {groups.map((g) => <MenuItem key={g.id} value={`group:${g.id}`}>{g.name}</MenuItem>)}
                    {sections.length > 0 && <ListSubheader>{t('events.fields.sections')}</ListSubheader>}
                    {sections.map((s) => <MenuItem key={s.id} value={`section:${s.id}`}>{s.name}</MenuItem>)}
                    {roles.length > 0 && <ListSubheader>{t('events.fields.roles')}</ListSubheader>}
                    {roles.map((r) => <MenuItem key={r.id} value={`role:${r.id}`}>{r.name}</MenuItem>)}
                </TextField>
                <ToggleButtonGroup exclusive size="small" value={view} aria-label={t('events.view')}
                                   onChange={(_e, v: View | null) => v && setView(v)}>
                    <ToggleButton value="month">{t('events.views.month')}</ToggleButton>
                    <ToggleButton value="week">{t('events.views.week')}</ToggleButton>
                    <ToggleButton value="list">{t('events.views.list')}</ToggleButton>
                </ToggleButtonGroup>
            </Box>

            {view !== 'list' && (
                <Paper variant="outlined" sx={{ overflowX: 'auto' }}>
                    <Box display="grid" gridTemplateColumns="repeat(7, minmax(110px, 1fr))">
                        {weekdays.map((w) => (
                            <Box key={w} sx={{ p: 1, fontWeight: 600, fontSize: '0.85rem', color: 'text.secondary', borderBottom: 1, borderColor: 'divider' }}>{w}</Box>
                        ))}
                        {days.map((d) => {
                            const list = occurrences.filter((o) => coversDay(o, d));
                            const outside = view === 'month' && d.getMonth() !== anchor.getMonth();
                            const isToday = ymd(d) === today;
                            const shown = view === 'month' ? list.slice(0, 3) : list;
                            return (
                                <Box key={ymd(d)}
                                     onDoubleClick={canEdit ? () => setEditing({ date: d }) : undefined}
                                     sx={{ minHeight: view === 'month' ? 108 : 280, p: 0.75, borderRight: 1, borderBottom: 1, borderColor: 'divider',
                                           opacity: outside ? 0.5 : 1, display: 'flex', flexDirection: 'column', gap: 0.5, minWidth: 0 }}>
                                    <Typography variant="caption" sx={{
                                        fontWeight: isToday ? 700 : 500, color: isToday ? 'primary.main' : 'text.secondary',
                                    }}>
                                        {view === 'week' ? d.toLocaleDateString(i18n.language, { day: 'numeric', month: 'short' }) : d.getDate()}
                                    </Typography>
                                    {shown.map((o) => chip(o, true))}
                                    {list.length > shown.length && (
                                        <Typography variant="caption" color="text.secondary">
                                            {t('events.more', { count: list.length - shown.length })}
                                        </Typography>
                                    )}
                                </Box>
                            );
                        })}
                    </Box>
                </Paper>
            )}

            {view === 'list' && (
                <Box display="flex" flexDirection="column" gap={2}>
                    {occurrences.length === 0 && <Typography color="text.secondary">{t('events.empty')}</Typography>}
                    {Object.entries(occurrences.reduce<Record<string, EventOccurrence[]>>((acc, o) => {
                        const key = ymd(new Date(o.start)) < ymd(range.from) ? ymd(range.from) : ymd(new Date(o.start));
                        (acc[key] ??= []).push(o);
                        return acc;
                    }, {})).map(([day, list]) => (
                        <Box key={day}>
                            <Typography variant="subtitle2" sx={{ mb: 1 }}>
                                {new Date(`${day}T00:00`).toLocaleDateString(i18n.language, { weekday: 'long', day: 'numeric', month: 'long' })}
                            </Typography>
                            <Paper variant="outlined">
                                {list.map((o) => (
                                    <Box key={`${o.eventId}-${o.occurrenceDate}`} onClick={() => setSelected(o)}
                                         sx={{ display: 'flex', alignItems: 'center', gap: 2, px: 2, py: 1.25, cursor: 'pointer',
                                               '&:not(:last-of-type)': { borderBottom: 1, borderColor: 'divider' }, '&:hover': { bgcolor: 'action.hover' } }}>
                                        <Typography sx={{ width: 90, color: 'text.secondary', fontSize: '0.9rem' }}>{time(o)}</Typography>
                                        <Box flexGrow={1} minWidth={0}>
                                            <Typography fontWeight={600} noWrap>{o.title}</Typography>
                                            {o.location && <Typography variant="body2" color="text.secondary" noWrap>{o.location}</Typography>}
                                        </Box>
                                        {o.recurring && <Repeat fontSize="small" color="action" aria-label={t('events.recurring')} />}
                                        <Chip size="small" color={CATEGORY_COLOR[o.category]} label={t(`events.category.${o.category}`)} />
                                    </Box>
                                ))}
                            </Paper>
                        </Box>
                    ))}
                </Box>
            )}

            {selected && (
                <OccurrenceDialog occurrence={selected} canEdit={canEdit} onClose={() => setSelected(null)}
                                  onEdit={async () => {
                                      try {
                                          const ev = await fetchEvent(selected.eventId);
                                          setSelected(null);
                                          setEditing({ event: ev });
                                      } catch {
                                          addNotification(t('events.loadFailed'));
                                      }
                                  }}
                                  onExclude={() => {
                                      const o = selected;
                                      setSelected(null);
                                      void run(() => excludeOccurrence(o.eventId, o.occurrenceDate), t('events.excluded'));
                                  }}
                                  onDelete={() => {
                                      const o = selected;
                                      if (!window.confirm(t(o.recurring ? 'events.confirmDeleteSeries' : 'events.confirmDelete'))) return;
                                      setSelected(null);
                                      void run(() => deleteEvent(o.eventId));
                                  }}
                                  onDownload={() => void run(() => downloadEventIcs(selected.eventId, selected.title, selected.occurrenceDate))} />
            )}
            {editing && (
                <EventDialog event={editing.event} initialDate={editing.date} onClose={() => setEditing(null)}
                             onSave={async (input: EventInput) => {
                                 const id = editing.event?.id;
                                 setEditing(null);
                                 await run(() => (id ? updateEvent(id, input) : createEvent(input)));
                             }} />
            )}
        </Box>
    );
}

function OccurrenceDialog({ occurrence: o, canEdit, onClose, onEdit, onExclude, onDelete, onDownload }: {
    occurrence: EventOccurrence;
    canEdit: boolean;
    onClose: () => void;
    onEdit: () => void;
    onExclude: () => void;
    onDelete: () => void;
    onDownload: () => void;
}) {
    const { t, i18n } = useTranslation();
    const start = new Date(o.start);
    const end = new Date(o.end);
    const dateFmt: Intl.DateTimeFormatOptions = { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' };
    let when: string;
    if (o.allDay) {
        const last = addDays(end, -1);
        when = ymd(last) > ymd(start)
            ? `${start.toLocaleDateString(i18n.language, dateFmt)} – ${last.toLocaleDateString(i18n.language, dateFmt)}`
            : start.toLocaleDateString(i18n.language, dateFmt);
    } else {
        const tf: Intl.DateTimeFormatOptions = { hour: '2-digit', minute: '2-digit' };
        when = ymd(end) > ymd(start)
            ? `${start.toLocaleString(i18n.language, { ...dateFmt, ...tf })} – ${end.toLocaleString(i18n.language, { ...dateFmt, ...tf })}`
            : `${start.toLocaleDateString(i18n.language, dateFmt)}, ${start.toLocaleTimeString(i18n.language, tf)} – ${end.toLocaleTimeString(i18n.language, tf)}`;
    }
    const audience = [...o.groups, ...o.sections, ...o.roles];

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {o.title}
                {o.recurring && <Tooltip title={t('events.recurring')}><Repeat fontSize="small" color="action" /></Tooltip>}
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={1.5}>
                    <Typography>{when}</Typography>
                    {o.location && (
                        <Typography display="flex" alignItems="center" gap={0.5}><PlaceOutlined fontSize="small" />{o.location}</Typography>
                    )}
                    <Box display="flex" flexWrap="wrap" gap={0.5}>
                        <Chip size="small" color={CATEGORY_COLOR[o.category]} label={t(`events.category.${o.category}`)} />
                        {audience.length === 0
                            ? <Chip size="small" variant="outlined" label={t('events.wholeClub')} />
                            : audience.map((a) => <Chip key={a.id} size="small" variant="outlined" label={a.name} />)}
                    </Box>
                    {o.description && <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>{o.description}</Typography>}
                </Box>
            </DialogContent>
            <DialogActions sx={{ flexWrap: 'wrap', gap: 1 }}>
                <Button startIcon={<Download />} onClick={onDownload}>{t('events.downloadIcs')}</Button>
                <Box flexGrow={1} />
                {canEdit && <>
                    {o.recurring && <Button color="warning" onClick={onExclude}>{t('events.excludeOccurrence')}</Button>}
                    <Button color="error" onClick={onDelete}>{o.recurring ? t('events.deleteSeries') : t('events.delete')}</Button>
                    <Button variant="contained" onClick={onEdit}>{o.recurring ? t('events.editSeries') : t('events.edit')}</Button>
                </>}
                {!canEdit && <Button onClick={onClose}>{t('buttons.close')}</Button>}
            </DialogActions>
        </Dialog>
    );
}
