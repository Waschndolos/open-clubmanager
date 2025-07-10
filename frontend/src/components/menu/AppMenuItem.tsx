import {ListItemButton, ListItemIcon, ListItemText, Tooltip} from "@mui/material";
import {useNavigate} from 'react-router-dom';
import {ReactElement} from "react";

type MenuItemProps = {
    label: string,
    icon?: ReactElement
    link?: string,
    collapsed: boolean;
};

export default function AppMenuItem({label, icon, link, collapsed}: MenuItemProps) {
    const navigate = useNavigate();

    return (
        <ListItemButton onClick={() => link && navigate(link)}
            sx={{
                justifyContent: collapsed ? "center" : "flex-start",
                borderRadius: 2,
                transition: 'background 0.2s, color 0.2s',
                '&:hover': {
                    backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#f4f6f8',
                    color: (theme) => theme.palette.mode === 'dark' ? '#ffb703' : '#8ecae6',
                },
                '&.Mui-selected, &.Mui-selected:hover': {
                    backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#e0e7ef',
                    color: (theme) => theme.palette.mode === 'dark' ? '#8ecae6' : '#819A91',
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
                />
            )}
        </ListItemButton>
    );
}
