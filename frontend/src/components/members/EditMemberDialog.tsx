import { useState } from "react";
import { Member } from "./types";
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  Grid,
} from "@mui/material";
import { useTranslation } from "react-i18next";

type Props = {
    member: Member;
    onClose: () => void;
    onSave: (updated: Member) => void;
};

export function EditMemberDialog({ member, onClose, onSave }: Props) {
    const {t} = useTranslation();
    const [formData, setFormData] = useState<Member>({ ...member });

    const handleChange = (key: keyof Member, value: string) => {
        setFormData(prev => ({ ...prev, [key]: value }));
    };

    const editableKeys = Object.keys(member).filter(
        (key) => key !== "email" && key !== "id"
    ) as (keyof Member)[];
    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{t("members.edit.title")}</DialogTitle>
            <DialogContent>
                <Grid container spacing={2}>
                    {editableKeys.map((key) => (
                        <Grid item xs={12} sm={6} key={key as string}>
                            <TextField
                                fullWidth
                                label={t("members.table.header." + key)}
                                value={formData[key] ?? ""}
                                onChange={(e) => handleChange(key, e.target.value)}
                            />
                        </Grid>
                    ))}
                </Grid>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button onClick={() => onSave(formData)} variant="contained">{t("buttons.save")}</Button>
            </DialogActions>
        </Dialog>
    );
}