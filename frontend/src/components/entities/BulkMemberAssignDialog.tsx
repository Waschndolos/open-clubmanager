import {
    Box,
    Button,
    Checkbox,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    InputAdornment,
    List,
    ListItem,
    ListItemButton,
    ListItemText,
    TextField,
    Typography,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Member, MemberSummary } from "../../api/types";
import { fetchMembers } from "../../api/members";

type BulkMemberAssignDialogProps = {
    open: boolean;
    entityName: string;
    fetchAssignedMembers: () => Promise<MemberSummary[]>;
    onSave: (memberIds: number[]) => Promise<void>;
    onClose: () => void;
};

export function BulkMemberAssignDialog({
    open,
    entityName,
    fetchAssignedMembers,
    onSave,
    onClose,
}: BulkMemberAssignDialogProps) {
    const { t } = useTranslation();
    const [allMembers, setAllMembers] = useState<Member[]>([]);
    const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
    const [search, setSearch] = useState("");
    const [loading, setLoading] = useState(false);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (!open) return;

        setLoading(true);
        Promise.all([fetchMembers(), fetchAssignedMembers()])
            .then(([members, assigned]) => {
                setAllMembers(members);
                setSelectedIds(new Set(assigned.map((m) => m.id)));
            })
            .finally(() => setLoading(false));
    }, [open, fetchAssignedMembers]);

    const filteredMembers = useMemo(() => {
        const q = search.trim().toLowerCase();
        if (!q) return allMembers;
        return allMembers.filter(
            (m) =>
                m.firstName?.toLowerCase().includes(q) ||
                m.lastName?.toLowerCase().includes(q) ||
                String(m.number ?? "").includes(q)
        );
    }, [allMembers, search]);

    const toggleMember = (id: number) => {
        setSelectedIds((prev) => {
            const next = new Set(prev);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            await onSave(Array.from(selectedIds));
            onClose();
        } finally {
            setSaving(false);
        }
    };

    const handleClose = () => {
        setSearch("");
        onClose();
    };

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            maxWidth="sm"
            fullWidth
            slotProps={{
                paper: {
                    sx: {
                        borderRadius: 4,
                        boxShadow: (theme) => theme.custom.boxShadow,
                        transition: "background 0.3s",
                    },
                },
            }}
        >
            <DialogTitle>
                {t("entities.assignMembers.title", { entity: entityName })}
            </DialogTitle>
            <DialogContent>
                <TextField
                    fullWidth
                    size="small"
                    placeholder={t("entities.assignMembers.search")}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    sx={{ mb: 1 }}
                    slotProps={{
                        input: {
                            startAdornment: (
                                <InputAdornment position="start">
                                    <SearchIcon fontSize="small" />
                                </InputAdornment>
                            ),
                        },
                    }}
                />
                {loading ? (
                    <Box display="flex" justifyContent="center" py={3}>
                        <CircularProgress size={32} />
                    </Box>
                ) : filteredMembers.length === 0 ? (
                    <Typography color="text.secondary" py={2} textAlign="center">
                        {t("entities.assignMembers.noMembers")}
                    </Typography>
                ) : (
                    <List dense disablePadding sx={{ maxHeight: 400, overflowY: "auto" }}>
                        {filteredMembers.map((m) => (
                            <ListItem key={m.id} disablePadding>
                                <ListItemButton
                                    onClick={() => toggleMember(m.id)}
                                    dense
                                >
                                    <Checkbox
                                        edge="start"
                                        checked={selectedIds.has(m.id)}
                                        tabIndex={-1}
                                        disableRipple
                                        size="small"
                                    />
                                    <ListItemText
                                        primary={`${m.lastName}, ${m.firstName}`}
                                        secondary={m.number != null ? `#${m.number}` : undefined}
                                    />
                                </ListItemButton>
                            </ListItem>
                        ))}
                    </List>
                )}
                <Typography variant="caption" color="text.secondary" mt={1} display="block">
                    {t("entities.assignMembers.selected", { count: selectedIds.size })}
                </Typography>
            </DialogContent>
            <DialogActions>
                <Button onClick={handleClose} disabled={saving}>
                    {t("buttons.abort")}
                </Button>
                <Button
                    variant="contained"
                    onClick={handleSave}
                    disabled={saving || loading}
                >
                    {saving ? <CircularProgress size={18} /> : t("buttons.save")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
