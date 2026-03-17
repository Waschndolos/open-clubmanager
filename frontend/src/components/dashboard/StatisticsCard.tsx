import {Box, Paper, Typography} from '@mui/material';
import React from 'react';
import CakeIcon from '@mui/icons-material/Cake';
import GroupIcon from '@mui/icons-material/Group';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import ExitToAppIcon from '@mui/icons-material/ExitToApp';
import {useTranslation} from "react-i18next";
import { alpha } from '@mui/material/styles';

interface StatisticCardProps {
    id: number;
    value: string | number;
    details?: string[];
}

const CARD_ACCENTS: Record<number, { color: string }> = {
    1: { color: '#4F6AF5' },
    2: { color: '#F59E0B' },
    3: { color: '#EF4444' },
};

export default function StatisticCard({id, value, details}: StatisticCardProps) {
    const {t} = useTranslation();
    const accent = CARD_ACCENTS[id] ?? { color: '#4F6AF5' };

    function getIcon() {
        switch (id) {
            case 1:
                return <GroupIcon sx={{ fontSize: 22 }} />;
            case 2:
                return <CakeIcon sx={{ fontSize: 22 }} />;
            case 3:
                return <ExitToAppIcon sx={{ fontSize: 22 }} />;
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