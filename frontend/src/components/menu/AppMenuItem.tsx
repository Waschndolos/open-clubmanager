import {ListItemButton, ListItemIcon, ListItemText, Tooltip} from "@mui/material";
import {useNavigate, useMatch} from 'react-router-dom';
import {ReactElement} from "react";

type MenuItemProps = {
    label: string,
    icon?: ReactElement
    link?: string,
    collapsed: boolean;
};

export default function AppMenuItem({label, icon, link, collapsed}: MenuItemProps) {
    const navigate = useNavigate();
    const match = useMatch({ path: link ?? "", end: false });
    const isActive = Boolean(match);

    return (
        <ListItemButton
            onClick={() => link && navigate(link)}
            selected={isActive}
            sx={{
                justifyContent: collapsed ? "center" : "flex-start",
                minHeight: 40,
                px: collapsed ? 1 : 1.5,
                py: 0.75,
                borderRadius: 2,
                transition: 'background 0.2s, color 0.2s',
                '&:hover': {
                    backgroundColor: (theme) =>
                        theme.palette.mode === 'dark'
                            ? 'rgba(255,255,255,0.05)'
                            : 'rgba(0,0,0,0.04)',
                },
            }}
        >
            <Tooltip title={collapsed ? label : ""} placement="right" arrow>
                <ListItemIcon sx={{ minWidth: 0, justifyContent: "center", color: isActive ? 'primary.main' : 'text.secondary' }}>
                    {icon}
                </ListItemIcon>
            </Tooltip>
            {!collapsed && (
                <ListItemText
                    primary={label}
                    sx={{ ml: 1.5 }}
                    primaryTypographyProps={{
                        fontSize: '0.875rem',
                        fontWeight: isActive ? 600 : 500,
                        color: isActive ? 'primary.main' : 'text.primary',
                    }}
                />
            )}
        </ListItemButton>
    );
}
