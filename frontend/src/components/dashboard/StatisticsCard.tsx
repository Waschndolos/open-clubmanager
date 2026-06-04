import {Box, Paper, Typography} from '@mui/material';
import React from 'react';
import EventNoteIcon from '@mui/icons-material/EventNote';
import PeopleAltIcon from '@mui/icons-material/PeopleAlt';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import PersonRemoveIcon from '@mui/icons-material/PersonRemove';
import {useTranslation} from "react-i18next";
import { alpha, useTheme } from '@mui/material/styles';

interface StatisticCardProps {
    id: number;
    value: string | number;
    details?: string[];
}

export default function StatisticCard({id, value, details}: StatisticCardProps) {
    const {t} = useTranslation();
    const theme = useTheme();

    // Map card IDs to theme colors
    const getCardColor = () => {
        switch (id) {
            case 1:
                return theme.palette.primary.main;
            case 2:
                return theme.palette.info.main;
            case 3:
                return theme.palette.error.main;
            default:
                return theme.palette.primary.main;
        }
    };

    const accent = { color: getCardColor() };

    function getIcon() {
        switch (id) {
            case 1:
                return <PeopleAltIcon sx={{ fontSize: 22 }} />;
            case 2:
                return <EventNoteIcon sx={{ fontSize: 22 }} />;
            case 3:
                return <PersonRemoveIcon sx={{ fontSize: 22 }} />;
            default:
                return <HelpOutlineIcon sx={{ fontSize: 22 }} />;
        }
    }

    function getTitle(): string | null {
        switch (id) {
            case 1:
                return t("dashboard.membercount");
            case 2:
                return t("dashboard.upcommingBirthdays");
            case 3:
                return t("dashboard.membersExitingThisYear");
            default:
                return null;
        }
    }

    return (
        <Paper sx={{
            p: 3,
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
        }}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 2 }}>
                <Box>
                    <Typography
                        variant="subtitle2"
                        color="text.secondary"
                        fontWeight={500}
                        sx={{ fontSize: '0.75rem', mb: 1 }}
                    >
                        {getTitle()}
                    </Typography>
                    <Typography variant="h3" fontWeight={700} lineHeight={1}>
                        {value}
                    </Typography>
                </Box>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 40,
                    height: 40,
                    borderRadius: 2,
                    bgcolor: alpha(accent.color, 0.1),
                    color: accent.color,
                    flexShrink: 0,
                }}>
                    {getIcon()}
                </Box>
            </Box>

            {details && details.length > 0 && (
                <Box sx={{ mt: 'auto', pt: 1.5, borderTop: '1px solid', borderColor: 'divider' }}>
                    {details.map((detail, index) => (
                        <Typography key={index} variant="body2" color="text.secondary">
                            {detail}
                        </Typography>
                    ))}
                </Box>
            )}
        </Paper>
    );
}