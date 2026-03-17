import {Box, Paper, Typography} from '@mui/material';
import React from 'react';
import CakeIcon from '@mui/icons-material/Cake';
import GroupIcon from '@mui/icons-material/Group';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import ExitToAppIcon from '@mui/icons-material/ExitToApp';
import {useTranslation} from "react-i18next";

interface StatisticCardProps {
    id: number;
    value: string | number;
    details?: string[];
}

const CARD_ACCENT_COLORS: Record<number, { bg: string; dark: string }> = {
    1: { bg: '#819A91', dark: '#5a7a70' },
    2: { bg: '#e8a838', dark: '#c98b20' },
    3: { bg: '#e05a5a', dark: '#c03a3a' },
};

export default function StatisticCard({id, value, details}: StatisticCardProps) {
    const {t} = useTranslation();
    const accent = CARD_ACCENT_COLORS[id] ?? { bg: '#819A91', dark: '#5a7a70' };

    function getIcon() {
        switch (id) {
            case 1:
                return <GroupIcon sx={{ fontSize: 28 }} />;
            case 2:
                return <CakeIcon sx={{ fontSize: 28 }} />;
            case 3:
                return <ExitToAppIcon sx={{ fontSize: 28 }} />;
            default:
                return <HelpOutlineIcon sx={{ fontSize: 28 }} />;
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
            position: 'relative',
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            overflow: 'hidden',
        }}>
            {/* Accent bar at top */}
            <Box sx={{
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                height: 4,
                bgcolor: accent.bg,
                borderRadius: '14px 14px 0 0',
            }} />

            <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: 2, mt: 0.5 }}>
                <Typography
                    variant="subtitle2"
                    color="text.secondary"
                    fontWeight={500}
                    sx={{ textTransform: 'uppercase', letterSpacing: 0.5, fontSize: '0.7rem' }}
                >
                    {getTitle()}
                </Typography>
                <Box sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    width: 48,
                    height: 48,
                    borderRadius: 2,
                    bgcolor: accent.bg,
                    color: '#fff',
                    flexShrink: 0,
                }}>
                    {getIcon()}
                </Box>
            </Box>

            <Typography variant="h3" fontWeight={700} lineHeight={1} mb={1}>
                {value}
            </Typography>

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