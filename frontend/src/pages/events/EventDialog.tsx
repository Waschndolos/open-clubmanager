import { useEffect, useState } from 'react';
import {
    Autocomplete,
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    Grid,
    MenuItem,
    Switch,
    TextField,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { fetchGroups } from '../../api/groups';
import { fetchRoles } from '../../api/roles';
import { fetchSections } from '../../api/sections';
import {
    addDays,
    ClubEvent,
    EVENT_CATEGORIES,
    EVENT_RECURRENCES,
    EventCategory,
    EventInput,
    EventRecurrence,
    localTimeZone,
    NamedRef,
    ymd,
} from '../../api/events';

type Props = {
    /** The series to edit, or undefined for a new event. */
    event?: ClubEvent;
    /** Start day for a new event. */
    initialDate?: Date;
    onClose: () => void;
    onSave: (input: EventInput) => Promise<void>;
};

const pad = (n: number) => String(n).padStart(2, '0');
const localDateTime = (d: Date) => `${ymd(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`;

export default function EventDialog({ event, initialDate, onClose, onSave }: Props) {
    const { t } = useTranslation();
    const [groups, setGroups] = useState<NamedRef[]>([]);
    const [sections, setSections] = useState<NamedRef[]>([]);
    const [roles, setRoles] = useState<NamedRef[]>([]);

    const defaultStart = (() => {
        const d = initialDate ? new Date(initialDate) : new Date();
        d.setHours(18, 0, 0, 0);
        return d;
    })();
    const startDate = event ? new Date(event.start) : defaultStart;
    const endDate = event ? new Date(event.end) : new Date(defaultStart.getTime() + 90 * 60000);

    const [title, setTitle] = useState(event?.title ?? '');
    const [category, setCategory] = useState<EventCategory>(event?.category ?? 'training');
    const [allDay, setAllDay] = useState(event?.allDay ?? false);
    const [start, setStart] = useState(localDateTime(startDate));
    const [end, setEnd] = useState(localDateTime(endDate));
    // All-day events: the UI shows the last day, the API wants the day after.
    const [startDay, setStartDay] = useState(ymd(startDate));
    const [endDay, setEndDay] = useState(ymd(event?.allDay ? addDays(endDate, -1) : startDate));
    const [location, setLocation] = useState(event?.location ?? '');
    const [description, setDescription] = useState(event?.description ?? '');
    const [recurrence, setRecurrence] = useState<EventRecurrence>(event?.recurrence ?? 'none');
    const [until, setUntil] = useState(event?.recurrenceUntil ?? '');
    const [groupSel, setGroupSel] = useState<NamedRef[]>(event?.groups ?? []);
    const [sectionSel, setSectionSel] = useState<NamedRef[]>(event?.sections ?? []);
    const [roleSel, setRoleSel] = useState<NamedRef[]>(event?.roles ?? []);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        fetchGroups().then(setGroups).catch(() => undefined);
        fetchSections().then(setSections).catch(() => undefined);
        fetchRoles().then(setRoles).catch(() => undefined);
    }, []);

    const startValue = allDay ? new Date(`${startDay}T00:00`) : new Date(start);
    const endValue = allDay ? addDays(new Date(`${endDay}T00:00`), 1) : new Date(end);
    const timesValid = !Number.isNaN(startValue.getTime()) && !Number.isNaN(endValue.getTime()) && endValue >= startValue;
    const untilValid = recurrence === 'none' || !until || until >= ymd(startValue);
    const valid = title.trim() !== '' && timesValid && untilValid;

    const save = async () => {
        setSaving(true);
        try {
            await onSave({
                title: title.trim(), category, allDay, timeZone: event?.timeZone && event.timeZone !== 'UTC' ? event.timeZone : localTimeZone(),
                start: startValue.toISOString(), end: endValue.toISOString(),
                location: location.trim() || undefined, description: description.trim() || undefined,
                recurrence, recurrenceUntil: recurrence !== 'none' && until ? until : undefined,
                groupIds: groupSel.map((g) => g.id), sectionIds: sectionSel.map((s) => s.id), roleIds: roleSel.map((r) => r.id),
            });
        } finally {
            setSaving(false);
        }
    };

    const audience = (label: string, options: NamedRef[], value: NamedRef[], set: (v: NamedRef[]) => void) => (
        <Autocomplete multiple size="small" options={options} value={value} getOptionLabel={(o) => o.name}
                      isOptionEqualToValue={(a, b) => a.id === b.id} onChange={(_e, v) => set(v)}
                      renderInput={(p) => <TextField {...p} label={label} />} />
    );

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{event ? t('events.edit') : t('events.add')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField autoFocus required label={t('events.fields.title')} value={title} onChange={(e) => setTitle(e.target.value)} />
                    <Grid container spacing={2} alignItems="center">
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <TextField select fullWidth label={t('events.fields.category')} value={category}
                                       onChange={(e) => setCategory(e.target.value as EventCategory)}>
                                {EVENT_CATEGORIES.map((c) => <MenuItem key={c} value={c}>{t(`events.category.${c}`)}</MenuItem>)}
                            </TextField>
                        </Grid>
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <FormControlLabel control={<Switch checked={allDay} onChange={(e) => setAllDay(e.target.checked)} />}
                                              label={t('events.fields.allDay')} />
                        </Grid>
                        {allDay ? <>
                            <Grid size={{ xs: 12, sm: 6 }}>
                                <TextField fullWidth type="date" label={t('events.fields.startDay')} value={startDay}
                                           onChange={(e) => {
                                               setStartDay(e.target.value);
                                               if (endDay < e.target.value) setEndDay(e.target.value);
                                           }} slotProps={{ inputLabel: { shrink: true } }} />
                            </Grid>
                            <Grid size={{ xs: 12, sm: 6 }}>
                                <TextField fullWidth type="date" label={t('events.fields.endDay')} value={endDay} error={!timesValid}
                                           onChange={(e) => setEndDay(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                            </Grid>
                        </> : <>
                            <Grid size={{ xs: 12, sm: 6 }}>
                                <TextField fullWidth type="datetime-local" label={t('events.fields.start')} value={start}
                                           onChange={(e) => {
                                               // Keep the duration when the start moves.
                                               const duration = new Date(end).getTime() - new Date(start).getTime();
                                               setStart(e.target.value);
                                               const s = new Date(e.target.value);
                                               if (!Number.isNaN(s.getTime()) && duration >= 0) setEnd(localDateTime(new Date(s.getTime() + duration)));
                                           }} slotProps={{ inputLabel: { shrink: true } }} />
                            </Grid>
                            <Grid size={{ xs: 12, sm: 6 }}>
                                <TextField fullWidth type="datetime-local" label={t('events.fields.end')} value={end} error={!timesValid}
                                           helperText={!timesValid ? t('events.endBeforeStart') : undefined}
                                           onChange={(e) => setEnd(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                            </Grid>
                        </>}
                        <Grid size={{ xs: 12, sm: 6 }}>
                            <TextField select fullWidth label={t('events.fields.recurrence')} value={recurrence}
                                       onChange={(e) => setRecurrence(e.target.value as EventRecurrence)}>
                                {EVENT_RECURRENCES.map((r) => <MenuItem key={r} value={r}>{t(`events.recurrence.${r}`)}</MenuItem>)}
                            </TextField>
                        </Grid>
                        <Grid size={{ xs: 12, sm: 6 }}>
                            {recurrence !== 'none' && (
                                <TextField fullWidth type="date" label={t('events.fields.until')} value={until} error={!untilValid}
                                           helperText={t('events.untilHelp')} onChange={(e) => setUntil(e.target.value)}
                                           slotProps={{ inputLabel: { shrink: true } }} />
                            )}
                        </Grid>
                    </Grid>
                    <TextField label={t('events.fields.location')} value={location} onChange={(e) => setLocation(e.target.value)} />
                    <TextField multiline minRows={2} label={t('events.fields.description')} value={description}
                               onChange={(e) => setDescription(e.target.value)} />
                    {audience(t('events.fields.groups'), groups, groupSel, setGroupSel)}
                    {audience(t('events.fields.sections'), sections, sectionSel, setSectionSel)}
                    {audience(t('events.fields.roles'), roles, roleSel, setRoleSel)}
                    <Box sx={{ color: 'text.secondary', fontSize: '0.85rem' }}>{t('events.audienceHelp')}</Box>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" disabled={!valid || saving} onClick={() => void save()}>{t('buttons.save')}</Button>
            </DialogActions>
        </Dialog>
    );
}
