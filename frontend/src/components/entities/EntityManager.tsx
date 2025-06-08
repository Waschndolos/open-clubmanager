import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
    Typography,
} from "@mui/material";
import { useEffect, useState } from "react";
import { EntityManagerProps, NamedEntity } from "./types";

export function EntityManager<T extends NamedEntity>({
                                                         title,
                                                         fetchFn,
                                                         createFn,
                                                         labels = {},
                                                     }: EntityManagerProps<T>) {
    const [entities, setEntities] = useState<T[]>([]);
    const [open, setOpen] = useState(false);
    const [name, setName] = useState("");
    const [description, setDescription] = useState("");

    useEffect(() => {
        fetchFn().then(setEntities);
    }, [fetchFn]);

    const handleCreate = async () => {
        const newItem = await createFn({ name, description } as Omit<T, "id">);
        setEntities(prev => [...prev, newItem]);
        setName("");
        setDescription("");
        setOpen(false);
    };

    return (
        <Box>
            <Typography variant="h5" gutterBottom>{title}</Typography>
            <Button variant="contained" onClick={() => setOpen(true)}>
                {labels.createButton ?? "Neu anlegen"}
            </Button>

            <ul>
                {entities.map(e => (
                    <li key={e.id}>
                        {e.name} {e.description && `– ${e.description}`}
                    </li>
                ))}
            </ul>

            <Dialog open={open} onClose={() => setOpen(false)}>
                <DialogTitle>{title}</DialogTitle>
                <DialogContent>
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
                        value={description}
                        onChange={e => setDescription(e.target.value)}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpen(false)}>Abbrechen</Button>
                    <Button variant="contained" onClick={handleCreate}>Erstellen</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
