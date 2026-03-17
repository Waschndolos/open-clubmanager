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
                borderRadius: 2,
                mb: 0.5,
                transition: 'background 0.2s, color 0.2s',
                '&:hover': {
                    backgroundColor: (theme) =>
                        theme.palette.mode === 'dark'
                            ? 'rgba(255,255,255,0.08)'
                            : 'rgba(0,0,0,0.06)',
                },
                '&.Mui-selected': {
                    backgroundColor: (theme) =>
                        theme.palette.mode === 'dark'
                            ? `rgba(124,142,248,0.15)`
                            : `rgba(79,106,245,0.1)`,
                    color: 'primary.main',
                    '& .MuiListItemIcon-root': {
                        color: 'primary.main',
                    },
                },
                '&.Mui-selected:hover': {
                    backgroundColor: (theme) =>
                        theme.palette.mode === 'dark'
                            ? `rgba(124,142,248,0.22)`
                            : `rgba(79,106,245,0.15)`,
                },
            }}
        >
            <Tooltip title={collapsed ? label : ""} placement="right">
                <ListItemIcon sx={{ minWidth: 0, justifyContent: "center" }}>
                    {icon}
                </ListItemIcon>
            </Tooltip>
            {!collapsed && (
                <ListItemText
                    primary={label}
                    sx={{
                        whiteSpace: "normal",
                        wordBreak: "break-word",
                        ml: 2,
                    }}
                    primaryTypographyProps={{ sx: { fontWeight: isActive ? 600 : 400 } }}
                />
            )}
        </ListItemButton>
    );
}
