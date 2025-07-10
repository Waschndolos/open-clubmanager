import {Box, List, Typography} from "@mui/material";
import AppMenuItem from "./AppMenuItem";
import packageJson from '../../../package.json'
import {useTranslation} from "react-i18next";
import {Dashboard, Money, PermIdentity, Person2, Settings} from "@mui/icons-material";

type Props = {
    collapsed: boolean;
};

export default function AppMenu({ collapsed }: Props) {
    const { t } = useTranslation();
    return (
        <Box
            className="bg-white dark:bg-dark-100 shadow-md dark:shadow-soft-dark rounded-xl transition-colors duration-500"
            sx={{
                display: "flex",
                flexDirection: "column",
                height: "100%",
                width: "100%",
                padding: 2,
                boxShadow: 2,
            }}
        >
            <List sx={{ flexGrow: 1 }}>
                <AppMenuItem label={t("menu.dashboard")} icon={<Dashboard />} link="dashboard" collapsed={collapsed} />
                <AppMenuItem label={t("menu.members")} icon={<Person2 />} link="members" collapsed={collapsed} />
                <AppMenuItem label={t("menu.finance")} icon={<Money />} link="finance" collapsed={collapsed} />
                <AppMenuItem label={t("menu.entities")} icon={<PermIdentity />} link="entities" collapsed={collapsed} />
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