import {ListItemButton, ListItemIcon, ListItemText} from "@mui/material";
import {useNavigate} from 'react-router-dom';
import {ReactElement} from "react";

type MenuItemProps = {
    label: string,
    icon?: ReactElement
    link?: string
};

export default function AppMenuItem({label, icon, link}: MenuItemProps) {
    const navigate = useNavigate();

    const handleClick = () => {
        navigate(`${link}`)
    };

    return (
        <ListItemButton onClick={handleClick}>
            {icon && (
                <ListItemIcon>
                    {icon}
                </ListItemIcon>
            )}
            <ListItemText primary={label}/>
        </ListItemButton>
    );
}
