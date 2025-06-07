import {ListItemButton, ListItemText} from "@mui/material";
import {useNavigate} from 'react-router-dom';

type MenuItemProps = {
    label: string,
    onClick?: () => void,
    link?: string
};

export default function AppMenuItem({label, link}: MenuItemProps) {
    const navigate = useNavigate();

    const handleClick = () => {
        if (link === 'dashboard') navigate('/dashboard');
        else if (link === 'members') navigate('/members');
        else if (link === 'finance') navigate('/finance');
        else if (link === 'settings') navigate('/settings');
    };

    return (
        <ListItemButton onClick={handleClick}>
            <ListItemText primary={label}/>
        </ListItemButton>
    );
}
