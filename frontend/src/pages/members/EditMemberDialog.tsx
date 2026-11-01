import {
    Accordion, AccordionDetails, Alert, Box,
    AccordionSummary,
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
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import {useTranslation} from "react-i18next";
import React, {useEffect, useState} from "react";
import {Member, Role, Group, ClubSection, canEditData, canEditFinance} from "../../api/types";
import {fetchRoles} from "../../api/roles";
import {fetchGroups} from "../../api/groups";
import {fetchSections} from "../../api/sections";
import dayjs from "dayjs";
import "dayjs/locale/de";
import "dayjs/locale/fr";
import {DatePicker, LocalizationProvider} from "@mui/x-date-pickers";
import {AdapterDayjs} from "@mui/x-date-pickers/AdapterDayjs";
import {deDE, enUS, frFR} from "@mui/x-date-pickers/locales";
import {useCollaboration, usePermission} from "../../context/CollaborationContext";
import {useRoleName} from "../../components/collaboration/common";
import {FamilySection, PayerExitWarning, useMemberHousehold} from "./FamilySection";
import {useAuth} from "../../context/AuthContext";
import {fetchSepaMandate, saveSepaMandate} from "../../api/sepa";
import {assignmentValue, FeeModel, fetchFeeAssignment, fetchFeeModels, saveFeeAssignment, toAssignment} from "../../api/feeModels";
import {FeeAssignmentSelect} from "../finance/FeeModels";
import {isAxiosError} from "axios";

// Date picker texts (and with the dayjs locale the field format, e.g.
// DD.MM.YYYY) follow the UI language.
const pickerLocaleText = {
    de: deDE.components.MuiLocalizationProvider.defaultProps.localeText,
    en: enUS.components.MuiLocalizationProvider.defaultProps.localeText,
    fr: frFR.components.MuiLocalizationProvider.defaultProps.localeText,
};

function pickerLanguage(language: string | undefined): keyof typeof pickerLocaleText {
    const lang = (language ?? "de").slice(0, 2);
    return lang in pickerLocaleText ? lang as keyof typeof pickerLocaleText : "en";
}

type Props = {
    member: Member;
    onClose: () => void;
    /** Rejects when saving failed; the dialog then shows the error. */
    onSave: (updated: Member) => Promise<void> | void;
    isNew?: boolean;
    /** Opens another member, for the links to family members. */
    onOpenMember?: (id: string) => void;
};

export function EditMemberDialog({member, onClose, onSave, isNew, onOpenMember}: Props) {
    const {t, i18n} = useTranslation();
    const language = pickerLanguage(i18n.language);
    const [formData, setFormData] = useState<Member>({...member});
    const [saveError, setSaveError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);
    const [expandedPanel, setExpandedPanel] = useState<string | false>('personal');

    // In a collaboration club the device's role decides which member data is
    // shown and editable (UX F10): hidden areas are not rendered at all,
    // read-only areas are disabled. Logins without edit rights (canEditData)
    // see everything read-only.
    const {appRole} = useAuth();
    const editable = canEditData(appRole);
    const core = usePermission('membersCore');
    const contact = usePermission('membersContact');
    const bank = usePermission('membersBank');
    const coreRO = !editable || core !== 'write';
    const contactRO = !editable || contact !== 'write';
    const bankRO = !editable || bank !== 'write';
    const canSave = !coreRO || !contactRO || !bankRO;
    const {status} = useCollaboration();
    const roleName = useRoleName();
    const [allRoles, setAllRoles] = useState<Role[]>([]);
    const [allGroups, setAllGroups] = useState<Group[]>([]);
    const [allSections, setAllSections] = useState<ClubSection[]>([]);
    const household = useMemberHousehold(isNew ? '' : member.id);
    // The mandate reference has its own endpoint, so it can only be set on
    // members that already exist.
    const [mandateRef, setMandateRef] = useState<{ loaded?: string; value: string; error?: string }>({value: ''});
    // Likewise the fee model, which belongs to the finances: only roles that
    // manage finances change it.
    const finance = usePermission('finance');
    const feeRO = !canEditFinance(appRole) || finance !== 'write';
    const [feeModels, setFeeModels] = useState<FeeModel[]>([]);
    const [feeModel, setFeeModel] = useState<{ loaded?: string; value: string; error?: string }>({value: 'auto'});

    useEffect(() => {
        fetchRoles().then(setAllRoles);
        fetchGroups().then(setAllGroups);
        fetchSections().then(setAllSections);
    }, []);

    useEffect(() => {
        if (isNew || bank === 'none') return;
        fetchSepaMandate(member.id)
            .then((ref) => setMandateRef({loaded: ref ?? '', value: ref ?? ''}))
            .catch(() => undefined);
    }, [isNew, bank, member.id]);

    useEffect(() => {
        if (isNew || finance === 'none') return;
        Promise.all([fetchFeeModels(), fetchFeeAssignment(member.id)])
            .then(([models, assignment]) => {
                setFeeModels(models);
                const value = assignmentValue(assignment);
                setFeeModel({loaded: value, value});
            })
            .catch(() => undefined);
    }, [isNew, finance, member.id]);

    const saveFeeModel = async (): Promise<boolean> => {
        if (feeRO || feeModel.loaded === undefined || feeModel.value === feeModel.loaded) return true;
        try {
            await saveFeeAssignment(member.id, toAssignment(feeModel.value));
            return true;
        } catch {
            setFeeModel((prev) => ({...prev, error: t("members.feeModel.saveFailed")}));
            setExpandedPanel('membership');
            return false;
        }
    };

    const saveMandateRef = async (): Promise<boolean> => {
        if (bankRO || mandateRef.loaded === undefined || mandateRef.value.trim() === mandateRef.loaded) return true;
        try {
            await saveSepaMandate(member.id, mandateRef.value.trim());
            return true;
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            setMandateRef((prev) => ({...prev, error: t(code === 'MANDATE_REFERENCE_TAKEN'
                ? "members.mandateReference.taken" : "members.mandateReference.invalid")}));
            setExpandedPanel('bank');
            return false;
        }
    };

    const saveErrorMessage = (e: unknown): string => {
        if (isAxiosError(e)) {
            if (e.response?.status === 409) return t("members.saveErrors.duplicate");
            const message = e.response?.data?.error ?? e.message;
            return t("members.saveErrors.generic", {message});
        }
        return t("members.saveErrors.generic", {message: e instanceof Error ? e.message : String(e)});
    };

    const handleAccordionChange = (panel: string) => (_: React.SyntheticEvent, isExpanded: boolean) => {
        setExpandedPanel(isExpanded ? panel : false);
    };

    const handleChange = (key: keyof Member, value: unknown) => {
        setFormData((prev) => ({...prev, [key]: value}));
    };

    // A picked calendar day is sent as UTC midnight of that day. toISOString()
    // would send local midnight, which is the previous day in UTC for Berlin.
    const calendarDate = (date: dayjs.Dayjs | null) =>
        date ? `${date.format("YYYY-MM-DD")}T00:00:00.000Z` : undefined;
    // Reads the calendar day back; parsing the full UTC timestamp would show the previous day west of UTC.
    const pickedDay = (value?: string | Date) =>
        value ? dayjs(typeof value === "string" ? value.slice(0, 10) : value) : null;

    return (
        <LocalizationProvider dateAdapter={AdapterDayjs} adapterLocale={language}
                              localeText={pickerLocaleText[language]}>
            <Dialog open onClose={onClose} maxWidth="md" fullWidth>
                <DialogTitle>
                    {isNew ? t("members.dialogs.create.title") : t("members.dialogs.edit.title")}
                </DialogTitle>

                <DialogContent>
                    {saveError && <Alert severity="error" sx={{mb: 2}} onClose={() => setSaveError(null)}>{saveError}</Alert>}
                    <Accordion defaultExpanded expanded={expandedPanel === 'personal'} onChange={handleAccordionChange('personal')}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon/>}>
                            <Typography>{t("members.dialogs.sections.personal")}</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Grid container spacing={2}>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth required type="number"
                                                                        label={t("members.table.header.number")}
                                                                        value={formData.number ?? ''}
                                                                        slotProps={{htmlInput: {min: 1, step: 1}}}
                                                                        onChange={(e) => handleChange("number", e.target.value === '' ? undefined : Number(e.target.value))} disabled={coreRO}/></Grid>
                                {contact !== 'none' && <Grid size={{xs: 12, sm: 6}}><TextField fullWidth required
                                                                        label={t("members.table.header.email")}
                                                                        value={formData.email ?? ''}
                                                                        onChange={(e) => handleChange("email", e.target.value)} disabled={contactRO}/></Grid>}
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth required
                                                                        label={t("members.table.header.firstName")}
                                                                        value={formData.firstName ?? ''}
                                                                        onChange={(e) => handleChange("firstName", e.target.value)} disabled={coreRO}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth required
                                                                        label={t("members.table.header.lastName")}
                                                                        value={formData.lastName ?? ''}
                                                                        onChange={(e) => handleChange("lastName", e.target.value)} disabled={coreRO}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}>
                                    <DatePicker sx={{xs: 12, sm: 6}}
                                                label={t("members.table.header.birthday")}
                                                value={pickedDay(formData.birthday)}
                                                onChange={(date) => handleChange("birthday", calendarDate(date))}
                                                disabled={coreRO}
                                    />
                                </Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.comment")}
                                                                        value={formData.comment ?? ''}
                                                                        onChange={(e) => handleChange("comment", e.target.value)} disabled={coreRO}/></Grid>
                                {contact !== 'none' && <>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.phone")}
                                                                        value={formData.phone ?? ''}
                                                                        onChange={(e) => handleChange("phone", e.target.value)} disabled={contactRO}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.phoneMobile")}
                                                                        value={formData.phoneMobile ?? ''}
                                                                        onChange={(e) => handleChange("phoneMobile", e.target.value)} disabled={contactRO}/></Grid>
                                </>}
                            </Grid>
                        </AccordionDetails>
                    </Accordion>

                    {contact !== 'none' && <Accordion expanded={expandedPanel === 'address'} onChange={handleAccordionChange('address')}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon/>} >
                            <Typography>{t("members.dialogs.sections.address")}</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Box component="fieldset" disabled={contactRO} sx={{border: 0, p: 0, m: 0, minWidth: 0}}>
                            <Grid container spacing={2}>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.street")}
                                                                        value={formData.street ?? ''}
                                                                        onChange={(e) => handleChange("street", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.postalCode")}
                                                                        value={formData.postalCode ?? ''}
                                                                        onChange={(e) => handleChange("postalCode", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth label={t("members.table.header.city")}
                                                                        value={formData.city ?? ''}
                                                                        onChange={(e) => handleChange("city", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.state")}
                                                                        value={formData.state ?? ''}
                                                                        onChange={(e) => handleChange("state", e.target.value)}/></Grid>
                            </Grid>
                        </Box>
                        </AccordionDetails>
                    </Accordion>}

                    {bank !== 'none' && <Accordion expanded={expandedPanel === 'bank'} onChange={handleAccordionChange('bank')}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon/>}>
                            <Typography>{t("members.dialogs.sections.bank")}</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            {bank !== 'write' && <Typography variant="body2" color="text.secondary" sx={{mb: 1}}>{t("collaboration.readOnly", {role: roleName(status?.role)})}</Typography>}
                            <Box component="fieldset" disabled={bankRO} sx={{border: 0, p: 0, m: 0, minWidth: 0}}>
                            <Grid container spacing={2}>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.accountHolder")}
                                                                        value={formData.accountHolder ?? ''}
                                                                        onChange={(e) => handleChange("accountHolder", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth label={t("members.table.header.iban")}
                                                                        value={formData.iban ?? ''}
                                                                        onChange={(e) => handleChange("iban", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth label={t("members.table.header.bic")}
                                                                        value={formData.bic ?? ''}
                                                                        onChange={(e) => handleChange("bic", e.target.value)}/></Grid>
                                <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.table.header.bankName")}
                                                                        value={formData.bankName ?? ''}
                                                                        onChange={(e) => handleChange("bankName", e.target.value)}/></Grid>

                                <Grid size={{xs: 12, sm: 6}}>
                                    <DatePicker
                                        label={t("members.table.header.sepaMandateDate")}
                                        value={pickedDay(formData.sepaMandateDate)}
                                        onChange={(date) => handleChange("sepaMandateDate", calendarDate(date))}
                                    />
                                </Grid>
                                {!isNew && <Grid size={{xs: 12, sm: 6}}><TextField fullWidth
                                                                        label={t("members.mandateReference.label")}
                                                                        value={mandateRef.value}
                                                                        error={!!mandateRef.error}
                                                                        helperText={mandateRef.error ?? t("members.mandateReference.help")}
                                                                        slotProps={{htmlInput: {maxLength: 35}}}
                                                                        onChange={(e) => setMandateRef((prev) => ({...prev, value: e.target.value, error: undefined}))}/></Grid>}
                            </Grid>
                        </Box>
                        </AccordionDetails>
                    </Accordion>}


                    <Accordion expanded={expandedPanel === 'membership'} onChange={handleAccordionChange('membership')}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon/>}>
                            <Typography>{t("members.dialogs.sections.membership")}</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Box component="fieldset" disabled={coreRO} sx={{border: 0, p: 0, m: 0, minWidth: 0}}>
                            <Grid container spacing={2}>
                                <DatePicker
                                    label={t("members.table.header.entryDate")}
                                    value={pickedDay(formData.entryDate)}
                                    onChange={(date) => handleChange("entryDate", calendarDate(date))}
                                />
                                <DatePicker
                                    label={t("members.table.header.exitDate")}
                                    value={pickedDay(formData.exitDate)}
                                    onChange={(date) => handleChange("exitDate", calendarDate(date))}
                                />
                            </Grid>
                        </Box>
                            {!isNew && feeModel.loaded !== undefined && (
                                <Box sx={{mt: 2, maxWidth: 360}}>
                                    <FeeAssignmentSelect models={feeModels} value={feeModel.value} disabled={feeRO}
                                                         label={t("members.feeModel.label")}
                                                         onChange={(value) => setFeeModel((prev) => ({...prev, value, error: undefined}))}/>
                                    <Typography variant="caption" color={feeModel.error ? "error" : "text.secondary"}>
                                        {feeModel.error ?? t("members.feeModel.help")}
                                    </Typography>
                                </Box>
                            )}
                            <PayerExitWarning memberId={member.id} exitDate={formData.exitDate} household={household}/>
                        </AccordionDetails>
                    </Accordion>

                    {!isNew && (core !== 'none' || contact !== 'none') && (
                        <Accordion expanded={expandedPanel === 'family'} onChange={handleAccordionChange('family')}>
                            <AccordionSummary expandIcon={<ExpandMoreIcon/>}>
                                <Typography>{t("members.dialogs.sections.family")}</Typography>
                            </AccordionSummary>
                            <AccordionDetails>
                                <FamilySection member={formData} household={household} onOpenMember={onOpenMember}/>
                            </AccordionDetails>
                        </Accordion>
                    )}


                    <Accordion expanded={expandedPanel === 'entities'} onChange={handleAccordionChange('entities')}>
                        <AccordionSummary expandIcon={<ExpandMoreIcon/>}>
                            <Typography>{t("members.dialogs.sections.entities")}</Typography>
                        </AccordionSummary>
                        <AccordionDetails>
                            <Box component="fieldset" disabled={coreRO} sx={{border: 0, p: 0, m: 0, minWidth: 0}}>
                            <Grid container spacing={2} sx={{mt: 3}}>
                                <Grid size={{xs: 12, sm: 6}}>
                                    <Autocomplete
                                        multiple
                                        options={allRoles}
                                        getOptionLabel={(option) => option.name}
                                        value={formData.roles}
                                        onChange={(_, value) => handleChange("roles", value)}
                                        renderInput={(params) => (
                                            <TextField {...params} label={t("members.table.header.roles")}/>
                                        )}
                                        renderValue={(value, getTagProps) =>
                                            value.map((option, index) => (
                                                <Chip label={option.name} {...getTagProps({index})} key={option.id}/>
                                            ))
                                        }
                                    />
                                </Grid>
                                <Grid size={{xs: 12, sm: 6}}>
                                    <Autocomplete
                                        multiple
                                        options={allGroups}
                                        getOptionLabel={(option) => option.name}
                                        value={formData.groups}
                                        onChange={(_, value) => handleChange("groups", value)}
                                        renderInput={(params) => (
                                            <TextField {...params} label={t("members.table.header.groups")}/>
                                        )}
                                        renderValue={(value, getTagProps) =>
                                            value.map((option, index) => (
                                                <Chip label={option.name} {...getTagProps({index})} key={option.id}/>
                                            ))
                                        }
                                    />
                                </Grid>
                                <Grid size={{xs: 12, sm: 6}}>
                                    <Autocomplete
                                        multiple
                                        options={allSections}
                                        getOptionLabel={(option) => option.name}
                                        value={formData.sections}
                                        onChange={(_, value) => handleChange("sections", value)}
                                        renderInput={(params) => (
                                            <TextField {...params} label={t("members.table.header.sections")}/>
                                        )}
                                        renderValue={(value, getTagProps) =>
                                            value.map((option, index) => (
                                                <Chip label={option.name} {...getTagProps({index})} key={option.id}/>
                                            ))
                                        }
                                    />
                                </Grid>
                            </Grid>
                        </Box>
                        </AccordionDetails>
                    </Accordion>

                </DialogContent>

                <DialogActions>
                    <Button onClick={onClose}>{t("buttons.abort")}</Button>
                    <Button
                        variant="contained"
                        disabled={!canSave || saving}
                        onClick={async () => {
                            setSaveError(null);
                            if ((contact !== 'none' && !formData.email) || !formData.firstName?.trim() || !formData.lastName?.trim()) {
                                setSaveError(t("members.saveErrors.required"));
                                return;
                            }
                            if (!Number.isInteger(formData.number) || (formData.number ?? 0) < 1) {
                                setSaveError(t("members.saveErrors.invalidNumber"));
                                return;
                            }
                            if (!await saveMandateRef()) return;
                            if (!await saveFeeModel()) return;
                            const payload = {
                                ...formData,
                                roleIds: formData.roles?.map(r => r.id),
                                groupIds: formData.groups?.map(g => g.id),
                                sectionIds: formData.sections?.map(s => s.id),
                                entryDate: formData.entryDate || undefined,
                                exitDate: formData.exitDate || undefined,
                                birthday: formData.birthday || undefined,
                                sepaMandateDate: formData.sepaMandateDate || undefined,
                            };
                            setSaving(true);
                            try {
                                await onSave(payload as Member);
                            } catch (e) {
                                setSaveError(saveErrorMessage(e));
                            } finally {
                                setSaving(false);
                            }
                        }}
                    >
                        {t("buttons.save")}
                    </Button>
                </DialogActions>
            </Dialog>
        </LocalizationProvider>
    );
}
