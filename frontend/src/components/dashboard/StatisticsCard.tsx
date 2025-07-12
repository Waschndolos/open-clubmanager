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

export default function StatisticCard({id, value, details}: StatisticCardProps) {
    const {t} = useTranslation();

    function getIcon() {
        switch (id) {
            case 1:
                return <GroupIcon fontSize="large"/>;
            case 2:
                return <CakeIcon fontSize="large"/>;
            case 3:
                return <ExitToAppIcon fontSize="large"/>;
            default:
                return <HelpOutlineIcon fontSize="large"/>;
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
            justifyContent: 'start',
            height: '100%',
        }}>
            <Box sx={{display: 'flex', alignItems: 'center', mb: 2}}>
                {getIcon()}
                <Typography variant="h6" sx={{ml: 2}}>{getTitle()}</Typography>
            </Box>
            <Typography variant="h4">{value}</Typography>
            {details && details.length > 0 && (
                <Box sx={{mt: 2}}>
                    <Typography variant="subtitle1">Details:</Typography>
                    <ul>
                        {details.map((detail, index) => (
                            <li key={index}>
                                <Typography variant="body2">{detail}</Typography>
                            </li>
                        ))}
                    </ul>
                </Box>
            )}
        </Paper>
    );
}