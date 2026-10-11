import { ListItemButton, ListItemIcon, ListItemText, Tooltip } from "@mui/material";
import { NavLink, useMatch } from "react-router-dom";
import { ReactElement } from "react";

type MenuItemProps = {
    label: string;
    icon: ReactElement;
    link: string;
    collapsed: boolean;
    onNavigate?: () => void;
};

export default function AppMenuItem({ label, icon, link, collapsed, onNavigate }: MenuItemProps) {
    const isActive = Boolean(useMatch({ path: link, end: false }));

    return (
        <Tooltip title={collapsed ? label : ""} placement="right" arrow disableInteractive>
            <ListItemButton
                component={NavLink}
                to={link}
                onClick={onNavigate}
                selected={isActive}
                aria-current={isActive ? "page" : undefined}
                aria-label={collapsed ? label : undefined}
                sx={{
                    position: "relative",
                    justifyContent: collapsed ? "center" : "flex-start",
                    minHeight: 40,
                    px: collapsed ? 0 : 1.25,
                    py: 0.5,
                    color: "text.secondary",
                    "&:hover": { color: "text.primary" },
                    "&.Mui-selected::before": {
                        content: '""',
                        position: "absolute",
                        left: collapsed ? 4 : -12,
                        top: 10,
                        bottom: 10,
                        width: 3,
                        borderRadius: 2,
                        bgcolor: "primary.main",
                    },
                }}
            >
                <ListItemIcon sx={{ minWidth: 0, color: "inherit", "& svg": { fontSize: 20 } }}>{icon}</ListItemIcon>
                {!collapsed && (
                    <ListItemText
                        primary={label}
                        sx={{ ml: 1.5, my: 0 }}
                        slotProps={{ primary: { noWrap: true, fontSize: "0.9rem", fontWeight: isActive ? 650 : 500 } }}
                    />
                )}
            </ListItemButton>
        </Tooltip>
    );
}
