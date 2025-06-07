import {Box, List, Typography} from "@mui/material";
import AppMenuItem from "./AppMenuItem";
import packageJson from '../../../package.json'
import {useTranslation} from "react-i18next";
import {Dashboard, Money, Person2, Settings} from "@mui/icons-material";


export default function AppMenu() {
    const { t } = useTranslation();
    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            width: '100%',
            padding: 4,
            boxShadow: 2,
        }}>
            <List sx={{ flexGrow: 1 }}>
                <AppMenuItem label={t('menu.dashboard')} icon={<Dashboard />} link="dashboard"/>
                <AppMenuItem label={t('menu.members')} icon={<Person2/>} link="members"/>
                <AppMenuItem label={t('menu.finance')} icon={<Money/>} link="finance"/>
                <AppMenuItem label={t('menu.settings')} icon={<Settings/>} link="settings"/>
            </List>

            <Box sx={{ marginTop: 'auto' }}>
                <Typography variant="body2" color="text.secondary" align="center">
                    Version {packageJson.version}
                </Typography>
            </Box>
        </Box>
    )
}