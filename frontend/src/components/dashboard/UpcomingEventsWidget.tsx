import { Paper, Stack, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import dayjs from "dayjs";
import { fetchEvents } from "../../api/events";
import { Event } from "../../api/types";
import { useTranslation } from "react-i18next";

export default function UpcomingEventsWidget() {
    const { t } = useTranslation();
    const [events, setEvents] = useState<Event[]>([]);

    useEffect(() => {
        const from = dayjs().startOf("day").toISOString();
        const to = dayjs().add(30, "day").endOf("day").toISOString();

        fetchEvents({ startDateFrom: from, startDateTo: to })
            .then((data) => setEvents(data.slice(0, 8)))
            .catch(() => setEvents([]));
    }, []);

    return (
        <Paper sx={{ p: 3, height: "100%" }}>
            <Typography variant="h6" fontWeight={700} sx={{ mb: 2 }}>
                {t("dashboard.nextEvents.title")}
            </Typography>
            <Stack spacing={1.5}>
                {events.length === 0 ? (
                    <Typography variant="body2" color="text.secondary">{t("dashboard.nextEvents.empty")}</Typography>
                ) : (
                    events.map((event) => (
                        <Stack key={event.id} spacing={0.3}>
                            <Typography variant="body2" fontWeight={700}>{event.title}</Typography>
                            <Typography variant="caption" color="text.secondary">
                                {dayjs(event.startDate).format("DD.MM.YYYY HH:mm")} · {t(`events.types.${event.type}`)}
                            </Typography>
                        </Stack>
                    ))
                )}
            </Stack>
        </Paper>
    );
}
