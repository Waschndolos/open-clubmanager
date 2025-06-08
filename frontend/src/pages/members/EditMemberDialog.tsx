import {useState} from "react";
import {Member} from "../../components/api/types";
import {
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    TextField,
} from "@mui/material";
import {useTranslation} from "react-i18next";
import Grid from "@mui/material/Grid";

type Props = {
    member: Member;
    onClose: () => void;
    onSave: (updated: Member) => void;
    isNew?: boolean;
};

export function EditMemberDialog({member, onClose, onSave, isNew}: Props) {
    const {t} = useTranslation();
    const [formData, setFormData] = useState<Member>({...member});

    console.log("received member", member)
    const handleChange = (key: keyof Member, value: string) => {
        setFormData(prev => ({...prev, [key]: value}));
    };

    const editableKeys = Object.keys(member).filter((key) => {
        if (key === "id") return false;
        return !(!isNew && key === "email");

    }) as (keyof Member)[];
    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {isNew ? t("members.dialogs.create.title") : t("members.dialogs.edit.title")}
            </DialogTitle>
            <DialogContent>
                <Grid container spacing={2}>
                    {editableKeys.map((key) => (
                        <Grid size={{xs: 12, sm: 6}} key={key as string}>
                            <TextField
                                fullWidth
                                required={key === "email"}
                                error={key === "email" && !formData.email}
                                helperText={key === "email" && !formData.email ? t("members.dialogs.validation.emailRequired") : ""}
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
                <Button onClick={() => {
                    if (!formData.email) return
                    onSave(formData)
                }

                } variant="contained">{t("buttons.save")}</Button>
            </DialogActions>
        </Dialog>
    );
}