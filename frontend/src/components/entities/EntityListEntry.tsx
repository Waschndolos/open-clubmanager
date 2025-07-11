import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    ListItem,
    ListItemText,
    TextField,
} from "@mui/material";
import { Edit, Delete } from "@mui/icons-material";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";

export type Entity = {
    id: number;
    name: string;
    description?: string;
};

type EntityListEntryProps<T extends Entity> = {
    item: T;
    onEdit: (item: T) => void;
    onDelete: (item: T) => void;
    labels?: {
        name?: string;
        description?: string;
    };
};

export function EntityListEntry<T extends Entity>({
                                                      item,
                                                      onEdit,
                                                      onDelete,
                                                      labels = {},
                                                  }: EntityListEntryProps<T>) {
    const { t } = useTranslation();
    const [open, setOpen] = useState(false);
    const [name, setName] = useState(item.name);
    const [description, setDescription] = useState(item.description ?? "");

    const handleSave = () => {
        onEdit({ ...item, name, description });
        setOpen(false);
    };

    return (
        <>
            <ListItem
                divider
                secondaryAction={
                    <Box>
                        <IconButton edge="end" onClick={() => setOpen(true)} aria-label="edit">
                            <Edit />
                        </IconButton>
                        <IconButton edge="end" onClick={() => onDelete(item)} aria-label="delete" sx={{ ml: 1 }}>
                            <Delete />
                        </IconButton>
                    </Box>
                }
                sx={{
                    borderRadius: 2,
                    mb: 1,
                    boxShadow: (theme) => theme.custom.boxShadow,
                    border: (theme) => theme.custom.border,
                    transition: 'background 0.3s, box-shadow 0.3s',
                }}
            >
                <ListItemText
                    primary={item.name}
                    secondary={item.description}
                    slotProps={{
                        primary: {
                            fontWeight: 500,
                        }
                    }}
                />
            </ListItem>

            <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
                <DialogTitle>{t("entities.edit")}</DialogTitle>
                <DialogContent>
                    <TextField
                        fullWidth
                        label={labels.name ?? t("entities.groups.dialogs.name")}
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        sx={{ mb: 2 }}
                    />
                    <TextField
                        fullWidth
                        label={labels.description ?? t("entities.groups.dialogs.description")}
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpen(false)}>{t("buttons.abort")}</Button>
                    <Button variant="contained" onClick={handleSave}>
                        {t("buttons.save")}
                    </Button>
                </DialogActions>
            </Dialog>
        </>
    );
}
