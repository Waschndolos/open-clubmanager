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
        <ListItemButton onClick={() => link && navigate(link)} sx={{ justifyContent: collapsed ? "center" : "flex-start" }}>
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
