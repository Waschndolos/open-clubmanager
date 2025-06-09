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
    Typography,
} from "@mui/material";
import { useTranslation } from "react-i18next";
import { useEffect, useState } from "react";
import { Member, Role, Group, ClubSection } from "../../components/api/types";
import { fetchRoles } from "../../components/api/roles";
import { fetchGroups } from "../../components/api/groups";
import { fetchSections } from "../../components/api/sections";

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

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>
                {isNew ? t("members.dialogs.create.title") : t("members.dialogs.edit.title")}
            </DialogTitle>

            <DialogContent>
                {/* 🔹 Persönliche Daten */}
                <Typography variant="subtitle1" sx={{ mt: 2 }}>{t("members.dialogs.sections.personal")}</Typography>
                <Grid container spacing={2}>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.number")} value={formData.number ?? ''} onChange={(e) => handleChange("number", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth required label={t("members.table.header.email")} value={formData.email ?? ''} onChange={(e) => handleChange("email", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.firstName")} value={formData.firstName ?? ''} onChange={(e) => handleChange("firstName", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.lastName")} value={formData.lastName ?? ''} onChange={(e) => handleChange("lastName", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.birthday")} value={formData.birthday ?? ''} onChange={(e) => handleChange("birthday", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.comment")} value={formData.comment ?? ''} onChange={(e) => handleChange("comment", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.phone")} value={formData.phone ?? ''} onChange={(e) => handleChange("phone", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.phoneMobile")} value={formData.phoneMobile ?? ''} onChange={(e) => handleChange("phoneMobile", e.target.value)} /></Grid>
                </Grid>

                {/* 🔹 Adress */}
                <Typography variant="subtitle1" sx={{ mt: 4 }}>{t("members.dialogs.sections.address")}</Typography>
                <Grid container spacing={2}>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.street")} value={formData.street ?? ''} onChange={(e) => handleChange("street", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.postalCode")} value={formData.postalCode ?? ''} onChange={(e) => handleChange("postalCode", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.city")} value={formData.city ?? ''} onChange={(e) => handleChange("city", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.state")} value={formData.state ?? ''} onChange={(e) => handleChange("state", e.target.value)} /></Grid>
                </Grid>

                {/* 🔹 Bank data */}
                <Typography variant="subtitle1" sx={{ mt: 4 }}>{t("members.dialogs.sections.bank")}</Typography>
                <Grid container spacing={2}>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.accountHolder")} value={formData.accountHolder ?? ''} onChange={(e) => handleChange("accountHolder", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.iban")} value={formData.iban ?? ''} onChange={(e) => handleChange("iban", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.bic")} value={formData.bic ?? ''} onChange={(e) => handleChange("bic", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.bankName")} value={formData.bankName ?? ''} onChange={(e) => handleChange("bankName", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.sepaMandateDate")} value={formData.sepaMandateDate ?? ''} onChange={(e) => handleChange("sepaMandateDate", e.target.value)} /></Grid>
                </Grid>

                {/* 🔹 club membership */}
                <Typography variant="subtitle1" sx={{ mt: 4 }}>{t("members.dialogs.sections.membership")}</Typography>
                <Grid container spacing={2}>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.entryDate")} value={formData.entryDate ?? ''} onChange={(e) => handleChange("entryDate", e.target.value)} /></Grid>
                    <Grid size={{xs:12, sm:6}}><TextField fullWidth label={t("members.table.header.exitDate")} value={formData.exitDate ?? ''} onChange={(e) => handleChange("exitDate", e.target.value)} /></Grid>
                </Grid>

                {/* 🔹 Roles / Groups / Sections */}
                <Grid container spacing={2} sx={{ mt: 3 }}>
                    <Grid size={{xs:12, sm:6}}>
                        <Autocomplete
                            multiple
                            options={allRoles}
                            getOptionLabel={(option) => option.name}
                            value={formData.roles}
                            onChange={(_, value) => handleChange("roles", value)}
                            renderInput={(params) => (
                                <TextField {...params} label={t("members.table.header.roles")} />
                            )}
                            renderValue={(value, getTagProps) =>
                                value.map((option, index) => (
                                    <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                                ))
                            }
                        />
                    </Grid>
                    <Grid size={{xs:12, sm:6}}>
                        <Autocomplete
                            multiple
                            options={allGroups}
                            getOptionLabel={(option) => option.name}
                            value={formData.groups}
                            onChange={(_, value) => handleChange("groups", value)}
                            renderInput={(params) => (
                                <TextField {...params} label={t("members.table.header.groups")} />
                            )}
                            renderValue={(value, getTagProps) =>
                                value.map((option, index) => (
                                    <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                                ))
                            }
                        />
                    </Grid>
                    <Grid size={{xs:12, sm:6}}>
                        <Autocomplete
                            multiple
                            options={allSections}
                            getOptionLabel={(option) => option.name}
                            value={formData.sections}
                            onChange={(_, value) => handleChange("sections", value)}
                            renderInput={(params) => (
                                <TextField {...params} label={t("members.table.header.sections")} />
                            )}
                            renderValue={(value, getTagProps) =>
                                value.map((option, index) => (
                                    <Chip label={option.name} {...getTagProps({ index })} key={option.id} />
                                ))
                            }
                        />
                    </Grid>
                </Grid>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button
                    variant="contained"
                    onClick={() => {
                        if (!formData.email) return;
                        const payload = {
                            ...formData,
                            roleIds: formData.roles?.map(r => r.id),
                            groupIds: formData.groups?.map(g => g.id),
                            sectionIds: formData.sections?.map(s => s.id),
                        };
                        onSave(payload as Member);
                    }}
                >
                    {t("buttons.save")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
