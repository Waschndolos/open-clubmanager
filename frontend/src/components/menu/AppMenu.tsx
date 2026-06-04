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
                boxShadow: "none",
                background: "transparent",
            }}
        >
            <List sx={{ flexGrow: 1, gap: 0.5, display: "flex", flexDirection: "column" }}>
                <AppMenuItem label={t("menu.dashboard")} icon={<GridView />} link="dashboard" collapsed={collapsed} />
                <AppMenuItem label={t("menu.members")} icon={<ManageAccounts />} link="members" collapsed={collapsed} />
                <AppMenuItem label={t("menu.finance")} icon={<AccountBalance />} link="finance" collapsed={collapsed} />
                <AppMenuItem label={t("menu.entities")} icon={<Badge />} link="entities" collapsed={collapsed} />
                <AppMenuItem label={t("menu.history")} icon={<History />} link="history" collapsed={collapsed} />
                <AppMenuItem label={t("menu.settings")} icon={<Settings />} link="settings" collapsed={collapsed} />
            </List>

            <Box sx={{ marginTop: "auto", textAlign: "center", py: 2, px: 1.5 }}>
                {collapsed ? (
                    <Box sx={{ display: "flex", justifyContent: "center", gap: 0.25 }}>
                        {packageJson.version.split("").map((char, index) => (
                            <Typography key={index} variant="caption" color="text.secondary" sx={{ fontSize: "0.65rem", fontWeight: 600 }}>
                                {char}
                            </Typography>
                        ))}
                    </Box>
                ) : (
                    <Typography
                        variant="body2"
                        color="text.secondary"
                        noWrap
                        sx={{
                            fontSize: "0.75rem",
                            fontWeight: 600,
                            letterSpacing: "0.03em",
                            opacity: 0.7,
                        }}
                    >
                        v{packageJson.version}
                    </Typography>
                )}

            </Box>
        </Box>
    );
}