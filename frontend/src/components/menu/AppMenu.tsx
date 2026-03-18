import {Box, List, Typography} from "@mui/material";
import AppMenuItem from "./AppMenuItem";
import packageJson from '../../../package.json'
import {useTranslation} from "react-i18next";
import {AccountBalance, Badge, GridView, History, ManageAccounts, Settings} from "@mui/icons-material";

type Props = {
    collapsed: boolean;
};

export default function AppMenu({ collapsed }: Props) {
    const { t } = useTranslation();
    return (
        <Box
            sx={{
                display: "flex",
                flexDirection: "column",
                height: "100%",
                width: "100%",
                padding: 2,
                boxShadow: (theme) => theme.custom.boxShadow,
            }}
        >
            <List sx={{ flexGrow: 1 }}>
                <AppMenuItem label={t("menu.dashboard")} icon={<GridView />} link="dashboard" collapsed={collapsed} />
                <AppMenuItem label={t("menu.members")} icon={<ManageAccounts />} link="members" collapsed={collapsed} />
                <AppMenuItem label={t("menu.finance")} icon={<AccountBalance />} link="finance" collapsed={collapsed} />
                <AppMenuItem label={t("menu.entities")} icon={<Badge />} link="entities" collapsed={collapsed} />
                <AppMenuItem label={t("menu.history")} icon={<History />} link="history" collapsed={collapsed} />
                <AppMenuItem label={t("menu.settings")} icon={<Settings />} link="settings" collapsed={collapsed} />
            </List>

            <Box sx={{ marginTop: "auto", textAlign: "center", py: 1 }}>
                {collapsed ? (
                    <Box>
                        {packageJson.version.split("").map((char, index) => (
                            <Typography key={index} variant="caption" color="text.secondary">
                                {char}
                            </Typography>
                        ))}
                    </Box>
                ) : (
                    <Typography variant="body2" color="text.secondary" noWrap>
                        Version {packageJson.version}
                    </Typography>
                )}


            </Box>
        </Box>
    );
}