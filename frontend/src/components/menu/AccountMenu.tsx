import { useState } from "react";
import { Avatar, Box, ButtonBase, Divider, ListItemIcon, Menu, MenuItem, Tooltip, Typography } from "@mui/material";
import { alpha } from "@mui/material/styles";
import LogoutRoundedIcon from "@mui/icons-material/LogoutRounded";
import SettingsOutlined from "@mui/icons-material/SettingsOutlined";
import UnfoldMoreRoundedIcon from "@mui/icons-material/UnfoldMoreRounded";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAuth } from "../../context/AuthContext";
import { setAccessToken as setApiAccessToken } from "../../api/api";

function initials(email: string | null): string {
    if (!email) return "?";
    const name = email.split("@")[0];
    const parts = name.split(/[._-]+/).filter(Boolean);
    const letters = parts.length > 1 ? parts[0][0] + parts[1][0] : name.slice(0, 2);
    return letters.toUpperCase();
}

type Props = {
    collapsed: boolean;
    onNavigate?: () => void;
};

/** The signed-in user at the foot of the sidebar, with settings and sign-out. */
export default function AccountMenu({ collapsed, onNavigate }: Props) {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { email, appRole, setAccessToken } = useAuth();
    const [anchor, setAnchor] = useState<HTMLElement | null>(null);

    const role = appRole ? t(`nav.roles.${appRole}`) : "";

    const go = (path: string) => {
        setAnchor(null);
        onNavigate?.();
        navigate(path);
    };

    const handleLogout = () => {
        setAnchor(null);
        setApiAccessToken(null);
        setAccessToken(null);
        localStorage.removeItem("refreshToken");
        navigate("/login", { replace: true });
    };

    const avatar = (
        <Avatar
            sx={{
                width: 32,
                height: 32,
                fontSize: "0.8rem",
                fontWeight: 700,
                bgcolor: (theme) => alpha(theme.palette.primary.main, 0.14),
                color: "primary.main",
            }}
        >
            {initials(email)}
        </Avatar>
    );

    return (
        <>
            <Tooltip title={collapsed ? t("nav.account") : ""} placement="right" arrow>
                <ButtonBase
                    onClick={(e) => setAnchor(e.currentTarget)}
                    aria-label={t("nav.account")}
                    aria-haspopup="menu"
                    aria-expanded={Boolean(anchor)}
                    sx={{
                        width: "100%",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: collapsed ? "center" : "flex-start",
                        gap: 1.25,
                        p: collapsed ? 0.5 : 1,
                        borderRadius: 2,
                        textAlign: "left",
                        "&:hover": { bgcolor: "action.hover" },
                        "&.Mui-focusVisible": { outline: (theme) => `2px solid ${theme.palette.primary.main}` },
                    }}
                >
                    {avatar}
                    {!collapsed && (
                        <>
                            <Box sx={{ minWidth: 0, flexGrow: 1 }}>
                                <Typography noWrap sx={{ fontSize: "0.85rem", fontWeight: 600, lineHeight: 1.3 }}>
                                    {email ?? t("nav.account")}
                                </Typography>
                                {role && (
                                    <Typography noWrap sx={{ fontSize: "0.75rem", color: "text.secondary", lineHeight: 1.3 }}>
                                        {role}
                                    </Typography>
                                )}
                            </Box>
                            <UnfoldMoreRoundedIcon sx={{ fontSize: 18, color: "text.secondary" }} />
                        </>
                    )}
                </ButtonBase>
            </Tooltip>
            <Menu
                anchorEl={anchor}
                open={Boolean(anchor)}
                onClose={() => setAnchor(null)}
                anchorOrigin={{ vertical: "top", horizontal: collapsed ? "right" : "left" }}
                transformOrigin={{ vertical: "bottom", horizontal: "left" }}
                slotProps={{ paper: { sx: { minWidth: 240, mt: -1 } } }}
            >
                <Box sx={{ px: 2, py: 1.25 }}>
                    <Typography variant="caption" color="text.secondary">{t("nav.signedInAs")}</Typography>
                    <Typography sx={{ fontWeight: 600, fontSize: "0.9rem", wordBreak: "break-all" }}>{email}</Typography>
                </Box>
                <Divider />
                <MenuItem onClick={() => go("/settings")}>
                    <ListItemIcon><SettingsOutlined fontSize="small" /></ListItemIcon>
                    {t("nav.settings")}
                </MenuItem>
                <MenuItem onClick={handleLogout}>
                    <ListItemIcon><LogoutRoundedIcon fontSize="small" /></ListItemIcon>
                    {t("logout.tooltip")}
                </MenuItem>
            </Menu>
        </>
    );
}
