import { useEffect, useState } from "react";
import {
    Autocomplete,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Grid,
    TextField,
} from "@mui/material";
import { useTranslation } from "react-i18next";
import { Member, Role, Group, ClubSection } from "../../components/api/types";
import {fetchRoles} from "../../components/api/roles";
import {fetchGroups} from "../../components/api/groups";
import {fetchSections} from "../../components/api/sections";

type Props = {
    member: Member;
    onClose: () => void;
    onSave: (updated: Member) => void;
    isNew?: boolean;
};

export function EditMemberDialog({ member, onClose, onSave, isNew }: Props) {
    const { t } = useTranslation();
    const [formData, setFormData] = useState<Member>({ ...member });

    const [allRoles, setAllRoles] = useState<Role[]>([]);
    const [allGroups, setAllGroups] = useState<Group[]>([]);
    const [allSections, setAllSections] = useState<ClubSection[]>([]);

    useEffect(() => {

        fetchRoles().then(setAllRoles);
        fetchGroups().then(setAllGroups);
        fetchSections().then(setAllSections);
    }, []);

    const handleChange = (key: keyof Member, value: unknown) => {
        setFormData((prev) => ({ ...prev, [key]: value }));
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
                        <Grid size={{xs:12, sm:6 }} key={key}>
                            <TextField
                                fullWidth
                                required={key === "email"}
                                error={key === "email" && !formData.email}
                                helperText={
                                    key === "email" && !formData.email
                                        ? t("members.dialogs.validation.emailRequired")
                                        : ""
                                }
                                label={t("members.table.header." + key)}
                                value={formData[key] ?? ""}
                                onChange={(e) => handleChange(key, e.target.value)}
                            />
                        </Grid>
                    ))}
                </Grid>

                <Autocomplete
                    multiple
                    options={allRoles}
                    getOptionLabel={(option) => option.name}
                    value={formData.roles}
                    onChange={(_, value) => handleChange("roles", value)}
                    renderValue={(value, getTagProps) =>
                        value.map((option, index) => (
                            <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                        ))
                    }
                    renderInput={(params) => (
                        <TextField {...params} label={t("members.table.header.roles")} margin="normal" />
                    )}
                />

                <Autocomplete
                    multiple
                    options={allGroups}
                    getOptionLabel={(option) => option.name}
                    value={formData.groups}
                    onChange={(_, value) => handleChange("groups", value)}
                    renderValue={(value, getTagProps) =>
                        value.map((option, index) => (
                            <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                        ))
                    }
                    renderInput={(params) => (
                        <TextField {...params} label={t("members.table.header.groups")} margin="normal" />
                    )}
                />

                <Autocomplete
                    multiple
                    options={allSections}
                    getOptionLabel={(option) => option.name}
                    value={formData.sections}
                    onChange={(_, value) => handleChange("sections", value)}
                    renderValue={(value, getTagProps) =>
                        value.map((option, index) => (
                            <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                        ))
                    }
                    renderInput={(params) => (
                        <TextField {...params} label={t("members.table.header.sections")} margin="normal" />
                    )}
                />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button
                    variant="contained"
                    onClick={() => {
                        if (!formData.email) return;
                        onSave(formData);
                    }}
                >
                    {t("buttons.save")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
