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
                return <PeopleAltIcon sx={{ fontSize: 24 }} />;
            case 2:
                return <EventNoteIcon sx={{ fontSize: 24 }} />;
            case 3:
                return <PersonRemoveIcon sx={{ fontSize: 24 }} />;
            default:
                return <HelpOutlineIcon sx={{ fontSize: 24 }} />;
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
            transition: 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
            '&:hover': {
                transform: 'translateY(-2px)',
                boxShadow: theme.palette.mode === 'dark'
                    ? '0 12px 32px rgba(0, 212, 170, 0.12)'
                    : '0 10px 28px rgba(0, 212, 170, 0.1)',
            }
        }}>
            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 2 }}>
                <Box>
                    <Typography
                        variant="subtitle2"
                        color="text.secondary"
                        fontWeight={600}
                        sx={{ fontSize: '0.7rem', mb: 1, textTransform: 'uppercase', letterSpacing: '0.05em', opacity: 0.8 }}
                    >
                        {getTitle()}
                    </Typography>
                    <Typography variant="h3" fontWeight={800} lineHeight={1}>
                        {value}
                    </Typography>
                </Box>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 48,
                    height: 48,
                    borderRadius: '12px',
                    bgcolor: alpha(accent.color, 0.12),
                    color: accent.color,
                    flexShrink: 0,
                    transition: 'all 0.3s ease',
                }}>
                    {getIcon()}
                </Box>
            </Box>

            {details && details.length > 0 && (
                <Box sx={{ mt: 'auto', pt: 2, borderTop: '1px solid', borderColor: 'divider' }}>
                    {details.map((detail, index) => (
                        <Typography key={index} variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                            {detail}
                        </Typography>
                    ))}
                </Box>
            )}
        </Paper>
    );
}