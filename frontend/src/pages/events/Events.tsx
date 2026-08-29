import {
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    Grid,
    IconButton,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    Stack,
    Tab,
    Tabs,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    TextField,
    Typography,
} from "@mui/material";
import { Add, CalendarMonth, Delete, Edit, Group } from "@mui/icons-material";
import { useEffect, useMemo, useState } from "react";
import dayjs from "dayjs";
import PageHeader from "../../components/common/PageHeader";
import { useTranslation } from "react-i18next";
import { Event, EventAttendanceStatus, EventAttendee, EventType, Member } from "../../api/types";
import { createEvent, deleteEvent, fetchEventAttendees, fetchEvents, removeEventAttendee, updateEvent, upsertEventAttendees } from "../../api/events";
import { fetchMembers } from "../../api/members";
import { useNotification } from "../../context/NotificationContext";

type EventFormState = {
    title: string;
    description: string;
    location: string;
    startDate: string;
    endDate: string;
    type: EventType;
    maxParticipants: string;
};

const EVENT_TYPES: EventType[] = ["meeting", "training", "tournament", "social", "other"];
const ATTENDANCE_STATUSES: EventAttendanceStatus[] = ["registered", "attended", "absent"];

function toLocalDateTimeInput(value: string): string {
    return dayjs(value).format("YYYY-MM-DDTHH:mm");
}

function fromLocalDateTimeInput(value: string): string {
    return dayjs(value).toISOString();
}

function defaultFormValues(base?: Event): EventFormState {
    if (!base) {
        const now = dayjs().minute(0).second(0);
        return {
            title: "",
            description: "",
            location: "",
            startDate: now.format("YYYY-MM-DDTHH:mm"),
            endDate: now.add(2, "hour").format("YYYY-MM-DDTHH:mm"),
            type: "meeting",
            maxParticipants: "",
        };
    }

    return {
        title: base.title,
        description: base.description ?? "",
        location: base.location ?? "",
        startDate: toLocalDateTimeInput(base.startDate),
        endDate: toLocalDateTimeInput(base.endDate),
        type: base.type,
        maxParticipants: base.maxParticipants ? String(base.maxParticipants) : "",
    };
}

export default function Events() {
    const { t } = useTranslation();
    const { addNotification } = useNotification();

    const [events, setEvents] = useState<Event[]>([]);
    const [members, setMembers] = useState<Member[]>([]);
    const [loading, setLoading] = useState(true);

    const [startDateFrom, setStartDateFrom] = useState("");
    const [startDateTo, setStartDateTo] = useState("");
    const [typeFilter, setTypeFilter] = useState<EventType | "">("");
    const [viewMode, setViewMode] = useState<"list" | "calendar">("list");

    const [dialogOpen, setDialogOpen] = useState(false);
    const [editingEvent, setEditingEvent] = useState<Event | null>(null);
    const [formState, setFormState] = useState<EventFormState>(defaultFormValues());

    const [attendeeDialogEvent, setAttendeeDialogEvent] = useState<Event | null>(null);
    const [attendees, setAttendees] = useState<EventAttendee[]>([]);
    const [newAttendeeId, setNewAttendeeId] = useState<number | "">("");

    const loadEvents = async () => {
        const filters = {
            startDateFrom: startDateFrom ? fromLocalDateTimeInput(startDateFrom) : undefined,
            startDateTo: startDateTo ? fromLocalDateTimeInput(startDateTo) : undefined,
            type: typeFilter || undefined,
        };
        const data = await fetchEvents(filters);
        setEvents(data);
    };

    useEffect(() => {
        Promise.all([fetchEvents(), fetchMembers()])
            .then(([eventData, memberData]) => {
                setEvents(eventData);
                setMembers(memberData);
            })
            .finally(() => setLoading(false));
    }, []);

    const groupedCalendarEvents = useMemo(() => {
        const groups = new Map<string, Event[]>();
        for (const event of events) {
            const key = dayjs(event.startDate).format("YYYY-MM-DD");
            const existing = groups.get(key) ?? [];
            existing.push(event);
            groups.set(key, existing.sort((a, b) => dayjs(a.startDate).valueOf() - dayjs(b.startDate).valueOf()));
        }
        return Array.from(groups.entries()).sort(([a], [b]) => a.localeCompare(b));
    }, [events]);

    const openCreateDialog = () => {
        setEditingEvent(null);
        setFormState(defaultFormValues());
        setDialogOpen(true);
    };

    const openEditDialog = (event: Event) => {
        setEditingEvent(event);
        setFormState(defaultFormValues(event));
        setDialogOpen(true);
    };

    const handleSaveEvent = async () => {
        const payload = {
            title: formState.title,
            description: formState.description || null,
            location: formState.location || null,
            startDate: fromLocalDateTimeInput(formState.startDate),
            endDate: fromLocalDateTimeInput(formState.endDate),
            type: formState.type,
            maxParticipants: formState.maxParticipants ? Number(formState.maxParticipants) : null,
        };

        if (editingEvent) {
            await updateEvent(editingEvent.id, payload);
            addNotification(t("events.snack.updated"));
        } else {
            await createEvent(payload);
            addNotification(t("events.snack.created"));
        }

        setDialogOpen(false);
        await loadEvents();
    };

    const handleDeleteEvent = async (eventId: number) => {
        await deleteEvent(eventId);
        addNotification(t("events.snack.deleted"));
        await loadEvents();
    };

    const openAttendeesDialog = async (event: Event) => {
        setAttendeeDialogEvent(event);
        const eventAttendees = await fetchEventAttendees(event.id);
        setAttendees(eventAttendees);
    };

    const handleAddAttendee = async () => {
        if (!attendeeDialogEvent || newAttendeeId === "") {
            return;
        }

        await upsertEventAttendees(attendeeDialogEvent.id, [{ memberId: newAttendeeId, status: "registered" }]);
        setAttendees(await fetchEventAttendees(attendeeDialogEvent.id));
        setNewAttendeeId("");
    };

    const handleAttendeeStatusChange = async (memberId: number, status: EventAttendanceStatus) => {
        if (!attendeeDialogEvent) {
            return;
        }
        await upsertEventAttendees(attendeeDialogEvent.id, [{ memberId, status }]);
        setAttendees(await fetchEventAttendees(attendeeDialogEvent.id));
    };

    const handleRemoveAttendee = async (memberId: number) => {
        if (!attendeeDialogEvent) {
            return;
        }
        await removeEventAttendee(attendeeDialogEvent.id, memberId);
        setAttendees(await fetchEventAttendees(attendeeDialogEvent.id));
    };

    const memberNameById = (memberId: number): string => {
        const member = members.find((entry) => entry.id === memberId);
        if (!member) {
            return String(memberId);
        }
        const firstName = member.firstName ?? "";
        const lastName = member.lastName ?? "";
        return `${firstName} ${lastName}`.trim() || String(memberId);
    };

    if (loading) {
        return <Typography>{t("events.loading")}</Typography>;
    }

    return (
        <Box p={3}>
            <PageHeader
                title={t("events.title")}
                icon={<CalendarMonth fontSize="small" />}
                actions={
                    <Button variant="contained" startIcon={<Add />} onClick={openCreateDialog}>
                        {t("events.create")}
                    </Button>
                }
            />

            <Stack direction={{ xs: "column", md: "row" }} spacing={2} sx={{ mb: 2 }}>
                <TextField
                    label={t("events.filters.startFrom")}
                    type="datetime-local"
                    value={startDateFrom}
                    onChange={(e) => setStartDateFrom(e.target.value)}
                    slotProps={{ inputLabel: { shrink: true } }}
                />
                <TextField
                    label={t("events.filters.startTo")}
                    type="datetime-local"
                    value={startDateTo}
                    onChange={(e) => setStartDateTo(e.target.value)}
                    slotProps={{ inputLabel: { shrink: true } }}
                />
                <FormControl sx={{ minWidth: 220 }}>
                    <InputLabel id="event-type-filter-label">{t("events.filters.type")}</InputLabel>
                    <Select
                        labelId="event-type-filter-label"
                        value={typeFilter}
                        label={t("events.filters.type")}
                        onChange={(e) => setTypeFilter(e.target.value as EventType | "")}
                    >
                        <MenuItem value="">{t("events.filters.allTypes")}</MenuItem>
                        {EVENT_TYPES.map((value) => (
                            <MenuItem value={value} key={value}>{t(`events.types.${value}`)}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
                <Button variant="outlined" onClick={loadEvents}>{t("events.filters.apply")}</Button>
            </Stack>

            <Tabs value={viewMode} onChange={(_, value: "list" | "calendar") => setViewMode(value)} sx={{ mb: 2 }}>
                <Tab value="list" label={t("events.views.list")} />
                <Tab value="calendar" label={t("events.views.calendar")} />
            </Tabs>

            {viewMode === "list" ? (
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>{t("events.table.title")}</TableCell>
                            <TableCell>{t("events.table.type")}</TableCell>
                            <TableCell>{t("events.table.location")}</TableCell>
                            <TableCell>{t("events.table.start")}</TableCell>
                            <TableCell>{t("events.table.end")}</TableCell>
                            <TableCell>{t("events.table.actions")}</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {events.map((event) => (
                            <TableRow key={event.id}>
                                <TableCell>{event.title}</TableCell>
                                <TableCell><Chip size="small" label={t(`events.types.${event.type}`)} /></TableCell>
                                <TableCell>{event.location ?? "-"}</TableCell>
                                <TableCell>{dayjs(event.startDate).format("DD.MM.YYYY HH:mm")}</TableCell>
                                <TableCell>{dayjs(event.endDate).format("DD.MM.YYYY HH:mm")}</TableCell>
                                <TableCell>
                                    <IconButton onClick={() => openEditDialog(event)}><Edit fontSize="small" /></IconButton>
                                    <IconButton onClick={() => openAttendeesDialog(event)}><Group fontSize="small" /></IconButton>
                                    <IconButton onClick={() => handleDeleteEvent(event.id)}><Delete fontSize="small" /></IconButton>
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            ) : (
                <Grid container spacing={2}>
                    {groupedCalendarEvents.map(([day, dayEvents]) => (
                        <Grid size={{ xs: 12, md: 6, lg: 4 }} key={day}>
                            <Paper sx={{ p: 2, height: "100%" }}>
                                <Typography variant="subtitle1" fontWeight={700} sx={{ mb: 1 }}>
                                    {dayjs(day).format("DD.MM.YYYY")}
                                </Typography>
                                <Stack spacing={1}>
                                    {dayEvents.map((event) => (
                                        <Box key={event.id} sx={{ border: "1px solid", borderColor: "divider", borderRadius: 1, p: 1.2 }}>
                                            <Typography variant="body2" fontWeight={700}>{event.title}</Typography>
                                            <Typography variant="caption" color="text.secondary">
                                                {dayjs(event.startDate).format("HH:mm")} - {dayjs(event.endDate).format("HH:mm")}
                                            </Typography>
                                        </Box>
                                    ))}
                                </Stack>
                            </Paper>
                        </Grid>
                    ))}
                </Grid>
            )}

            <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
                <DialogTitle>{editingEvent ? t("events.dialog.editTitle") : t("events.dialog.createTitle")}</DialogTitle>
                <DialogContent>
                    <Stack spacing={2} sx={{ mt: 1 }}>
                        <TextField label={t("events.dialog.title")} value={formState.title} onChange={(e) => setFormState({ ...formState, title: e.target.value })} />
                        <TextField label={t("events.dialog.description")} multiline minRows={3} value={formState.description} onChange={(e) => setFormState({ ...formState, description: e.target.value })} />
                        <TextField label={t("events.dialog.location")} value={formState.location} onChange={(e) => setFormState({ ...formState, location: e.target.value })} />
                        <TextField label={t("events.dialog.startDate")} type="datetime-local" slotProps={{ inputLabel: { shrink: true } }} value={formState.startDate} onChange={(e) => setFormState({ ...formState, startDate: e.target.value })} />
                        <TextField label={t("events.dialog.endDate")} type="datetime-local" slotProps={{ inputLabel: { shrink: true } }} value={formState.endDate} onChange={(e) => setFormState({ ...formState, endDate: e.target.value })} />
                        <FormControl>
                            <InputLabel id="event-type-label">{t("events.dialog.type")}</InputLabel>
                            <Select
                                labelId="event-type-label"
                                label={t("events.dialog.type")}
                                value={formState.type}
                                onChange={(e) => setFormState({ ...formState, type: e.target.value as EventType })}
                            >
                                {EVENT_TYPES.map((value) => (
                                    <MenuItem key={value} value={value}>{t(`events.types.${value}`)}</MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                        <TextField
                            label={t("events.dialog.maxParticipants")}
                            type="number"
                            value={formState.maxParticipants}
                            onChange={(e) => setFormState({ ...formState, maxParticipants: e.target.value })}
                        />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDialogOpen(false)}>{t("buttons.abort")}</Button>
                    <Button onClick={handleSaveEvent} variant="contained">{t("buttons.save")}</Button>
                </DialogActions>
            </Dialog>

            <Dialog open={Boolean(attendeeDialogEvent)} onClose={() => setAttendeeDialogEvent(null)} maxWidth="md" fullWidth>
                <DialogTitle>{t("events.attendees.title", { event: attendeeDialogEvent?.title ?? "" })}</DialogTitle>
                <DialogContent>
                    <Stack direction={{ xs: "column", md: "row" }} spacing={2} sx={{ mb: 2, mt: 1 }}>
                        <FormControl sx={{ minWidth: 280 }}>
                            <InputLabel id="attendee-member-label">{t("events.attendees.member")}</InputLabel>
                            <Select
                                labelId="attendee-member-label"
                                value={newAttendeeId}
                                label={t("events.attendees.member")}
                                onChange={(e) => setNewAttendeeId(Number(e.target.value))}
                            >
                                {members.map((member) => (
                                    <MenuItem key={member.id} value={member.id}>
                                        {memberNameById(member.id)}
                                    </MenuItem>
                                ))}
                            </Select>
                        </FormControl>
                        <Button variant="outlined" onClick={handleAddAttendee}>
                            {t("events.attendees.add")}
                        </Button>
                    </Stack>

                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t("events.attendees.member")}</TableCell>
                                <TableCell>{t("events.attendees.status")}</TableCell>
                                <TableCell>{t("events.table.actions")}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {attendees.map((attendee) => (
                                <TableRow key={attendee.memberId}>
                                    <TableCell>{attendee.member ? `${attendee.member.firstName} ${attendee.member.lastName}` : memberNameById(attendee.memberId)}</TableCell>
                                    <TableCell>
                                        <Select
                                            size="small"
                                            value={attendee.status}
                                            onChange={(e) => handleAttendeeStatusChange(attendee.memberId, e.target.value as EventAttendanceStatus)}
                                        >
                                            {ATTENDANCE_STATUSES.map((status) => (
                                                <MenuItem key={status} value={status}>{t(`events.attendance.${status}`)}</MenuItem>
                                            ))}
                                        </Select>
                                    </TableCell>
                                    <TableCell>
                                        <IconButton onClick={() => handleRemoveAttendee(attendee.memberId)}>
                                            <Delete fontSize="small" />
                                        </IconButton>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setAttendeeDialogEvent(null)}>{t("buttons.abort")}</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
