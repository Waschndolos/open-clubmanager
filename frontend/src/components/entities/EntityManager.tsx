import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    List,
    TextField,
    Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { EntityManagerProps, NamedEntity } from "./types";
import { useTranslation } from "react-i18next";
import { EntityListEntry } from "./EntityListEntry";

export function EntityManager<T extends NamedEntity>({
                                                         description,
                                                         fetchFn,
                                                         createFn,
                                                         updateFn,
                                                         deleteFn,
                                                         labels = {},
                                                     }: EntityManagerProps<T>) {
    const { t } = useTranslation();
    const [entities, setEntities] = useState<T[]>([]);
    const [open, setOpen] = useState(false);
    const [name, setName] = useState("");
    const [entityDescription, setEntityDescription] = useState("");

    useEffect(() => {
        fetchFn().then(setEntities);
    }, [fetchFn]);

    const handleCreate = async () => {
        const newItem = await createFn({ name, description: entityDescription } as Omit<T, "id">);
        setEntities(prev => [...prev, newItem]);
        setName("");
        setEntityDescription("");
        setOpen(false);
    };

    const handleDelete = async (item: T) => {
        await deleteFn(item);
        const refreshed = await fetchFn();
        setEntities(refreshed);
    };

    const handleUpdate = async (item: T) => {
        await updateFn(item);
        const refreshed = await fetchFn();
        setEntities(refreshed);
    };

    return (
        <Box display="flex" flexDirection="column" gap={3}>
            <Box>

                {description && (
                    <Typography variant="body1" color="text.secondary">
                        {description}
                    </Typography>
                )}
            </Box>

            <Box display="flex" justifyContent="flex-end">
                <Button variant="contained" onClick={() => setOpen(true)}>
                    {labels.createButton ?? t("buttons.save")}
                </Button>
            </Box>

            <List sx={{
                backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#181926' : '#f4f6f8',
                borderRadius: 3,
                boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 2px 12px 0 rgba(142,202,230,0.10)' : '0 1px 4px 0 rgba(0,0,0,0.04)',
                p: 2,
                transition: 'background 0.3s',
            }}>
                {entities.map(e => (
                    <EntityListEntry
                        key={e.id}
                        item={e}
                        onEdit={handleUpdate}
                        onDelete={handleDelete}
                    />
                ))}
            </List>

            <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth
                PaperProps={{
                    sx: {
                        backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#fff',
                        borderRadius: 4,
                        boxShadow: (theme) => theme.palette.mode === 'dark' ? '0 4px 24px 0 rgba(142,202,230,0.15)' : '0 2px 8px 0 rgba(0,0,0,0.10)',
                        transition: 'background 0.3s',
                    }
                }}
            >
                <DialogTitle>{labels.createButton ?? t("entities.create")}</DialogTitle>
                <DialogContent sx={{ pt: 2 }}>
                    <TextField
                        fullWidth
                        label={labels.name ?? "Name"}
                        value={name}
                        onChange={e => setName(e.target.value)}
                        sx={{ mb: 2 }}
                    />
                    <TextField
                        fullWidth
                        label={labels.description ?? "Beschreibung"}
                        value={entityDescription}
                        onChange={e => setEntityDescription(e.target.value)}
                        multiline
                        minRows={2}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpen(false)}>{t("buttons.abort")}</Button>
                    <Button variant="contained" onClick={handleCreate}>{t("buttons.save")}</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
