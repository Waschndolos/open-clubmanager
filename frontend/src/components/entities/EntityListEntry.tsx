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
    Tooltip,
} from "@mui/material";
import { Edit, Delete, PeopleAlt } from "@mui/icons-material";
import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { MemberSummary } from "../../api/types";
import { BulkMemberAssignDialog } from "./BulkMemberAssignDialog";

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
    fetchMembersFn?: (id: number) => Promise<MemberSummary[]>;
    assignMembersFn?: (id: number, memberIds: number[]) => Promise<void>;
};

export function EntityListEntry<T extends Entity>({
    item,
    onEdit,
    onDelete,
    labels = {},
    fetchMembersFn,
    assignMembersFn,
}: EntityListEntryProps<T>) {
    const { t } = useTranslation();
    const [editOpen, setEditOpen] = useState(false);
    const [assignOpen, setAssignOpen] = useState(false);
    const [name, setName] = useState(item.name);
    const [description, setDescription] = useState(item.description ?? "");

    const handleSave = () => {
        onEdit({ ...item, name, description });
        setEditOpen(false);
    };

    const showMembersButton = fetchMembersFn != null && assignMembersFn != null;

    return (
        <>
            <ListItem
                divider
                secondaryAction={
                    <Box>
                        {showMembersButton && (
                            <Tooltip title={t("entities.assignMembers.tooltip")}>
                                <IconButton
                                    edge="end"
                                    onClick={() => setAssignOpen(true)}
                                    aria-label="assign members"
                                    sx={{ mr: 0.5 }}
                                >
                                    <PeopleAlt />
                                </IconButton>
                            </Tooltip>
                        )}
                        <IconButton edge="end" onClick={() => setEditOpen(true)} aria-label="edit">
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
                    transition: "background 0.3s, box-shadow 0.3s",
                }}
            >
                <ListItemText
                    primary={item.name}
                    secondary={item.description}
                    slotProps={{
                        primary: {
                            fontWeight: 500,
                        },
                    }}
                />
            </ListItem>

            <Dialog open={editOpen} onClose={() => setEditOpen(false)} maxWidth="sm" fullWidth>
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
                    <Button onClick={() => setEditOpen(false)}>{t("buttons.abort")}</Button>
                    <Button variant="contained" onClick={handleSave}>
                        {t("buttons.save")}
                    </Button>
                </DialogActions>
            </Dialog>

            {showMembersButton && (
                <BulkMemberAssignDialog
                    open={assignOpen}
                    entityName={item.name}
                    fetchAssignedMembers={() => fetchMembersFn!(item.id)}
                    onSave={(memberIds) => assignMembersFn!(item.id, memberIds)}
                    onClose={() => setAssignOpen(false)}
                />
            )}
        </>
    );
}

