import { Box, IconButton, List, Tooltip, Typography } from "@mui/material";
import KeyboardDoubleArrowLeftRounded from "@mui/icons-material/KeyboardDoubleArrowLeftRounded";
import KeyboardDoubleArrowRightRounded from "@mui/icons-material/KeyboardDoubleArrowRightRounded";
import { useTranslation } from "react-i18next";
import packageJson from "../../../package.json";
import BrandMark from "../common/BrandMark";
import AppMenuItem from "./AppMenuItem";
import AccountMenu from "./AccountMenu";
import { useNavGroups } from "./navItems";

type Props = {
    collapsed: boolean;
    /** Desktop only: toggles the collapsed rail. Omitted in the mobile drawer. */
    onToggleCollapsed?: () => void;
    /** Mobile drawer: close after a navigation. */
    onNavigate?: () => void;
};

export default function AppMenu({ collapsed, onToggleCollapsed, onNavigate }: Props) {
    const { t } = useTranslation();
    const groups = useNavGroups();

    return (
        <Box
            component="nav"
            aria-label={t("nav.main")}
            sx={{ display: "flex", flexDirection: "column", height: "100%", width: "100%" }}
        >
            <Box
                sx={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: collapsed ? "center" : "flex-start",
                    height: 64,
                    px: collapsed ? 0 : 2.25,
                    flexShrink: 0,
                }}
            >
                <BrandMark compact={collapsed} size={30} />
            </Box>

            <Box sx={{ flexGrow: 1, overflowY: "auto", overflowX: "hidden", px: collapsed ? 1.25 : 1.75, pt: 1, scrollbarWidth: "thin" }}>
                {groups.map((group, index) => (
                    <Box key={group.label ?? index} sx={{ mb: 2 }}>
                        {group.label && (collapsed ? (
                            <Box sx={{ height: 1, bgcolor: "divider", mx: 1, mb: 1.5 }} role="separator" />
                        ) : (
                            <Typography
                                component="h2"
                                sx={{ px: 1.25, mb: 0.75, fontSize: "0.75rem", fontWeight: 600, color: "text.secondary" }}
                            >
                                {group.label}
                            </Typography>
                        ))}
                        <List disablePadding sx={{ display: "grid", gap: 0.25 }}>
                            {group.items.map((item) => (
                                <AppMenuItem key={item.link} {...item} collapsed={collapsed} onNavigate={onNavigate} />
                            ))}
                        </List>
                    </Box>
                ))}
            </Box>

            <Box sx={{ px: collapsed ? 1.25 : 1.75, pb: 1.5, pt: 1, borderTop: 1, borderColor: "divider", flexShrink: 0 }}>
                <AccountMenu collapsed={collapsed} onNavigate={onNavigate} />
                <Box
                    sx={{
                        display: "flex",
                        flexDirection: collapsed ? "column" : "row",
                        alignItems: "center",
                        justifyContent: "space-between",
                        gap: 0.5,
                        mt: 1,
                        pl: collapsed ? 0 : 1,
                    }}
                >
                    <Typography variant="caption" color="text.secondary" sx={{ fontSize: "0.7rem", opacity: 0.75 }}>
                        v{packageJson.version}
                    </Typography>
                    {onToggleCollapsed && (
                        <Tooltip title={collapsed ? t("nav.expand") : t("nav.collapse")} placement="right">
                            <IconButton
                                size="small"
                                onClick={onToggleCollapsed}
                                aria-label={collapsed ? t("nav.expand") : t("nav.collapse")}
                                sx={{ color: "text.secondary" }}
                            >
                                {collapsed
                                    ? <KeyboardDoubleArrowRightRounded fontSize="small" />
                                    : <KeyboardDoubleArrowLeftRounded fontSize="small" />}
                            </IconButton>
                        </Tooltip>
                    )}
                </Box>
            </Box>
        </Box>
    );
}
