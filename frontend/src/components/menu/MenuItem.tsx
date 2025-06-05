import { ListItemButton, ListItemText } from "@mui/material";

type MenuItemProps = {
    label: string;
    onClick?: () => void;
};

export default function MenuItem({ label, onClick }: MenuItemProps) {
    return (
        <ListItemButton onClick={onClick}>
            <ListItemText primary={label} />
        </ListItemButton>
    );
}
