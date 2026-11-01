import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Alert,
    Box,
    Button,
    Checkbox,
    Chip,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogContentText,
    DialogTitle,
    FormControl,
    FormControlLabel,
    IconButton,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { Add, Delete, Edit, PlaylistAddCheck, Preview } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import { isAxiosError } from 'axios';
import {
    assignmentValue,
    createFeeModel,
    createFeeRun,
    deleteFeeModel,
    deleteFeeRun,
    FEE_INTERVALS,
    FeeModel,
    FeeRun,
    FeeRunPlan,
    fetchFeeModels,
    fetchFeeRuns,
    fetchMemberFeeModels,
    MemberFeeModel,
    previewFeeRun,
    saveFeeAssignment,
    toAssignment,
    updateFeeModel,
} from '../../api/feeModels';
import { MemberFee } from '../../api/types';
import { useNotification } from '../../context/NotificationContext';

const money = (n: number) => `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €`;
const localDate = (ymd: string) => new Date(`${ymd.slice(0, 10)}T00:00:00`).toLocaleDateString();
const memberName = (m: { firstName: string; lastName: string; number: number }) => `${m.firstName} ${m.lastName} (${m.number})`;

function useAgeLabel() {
    const { t } = useTranslation();
    return (m: Pick<FeeModel, 'minAge' | 'maxAge'>) => {
        if (m.minAge !== undefined && m.maxAge !== undefined) return t('feeModels.ageRange', { min: m.minAge, max: m.maxAge });
        if (m.minAge !== undefined) return t('feeModels.ageFrom', { min: m.minAge });
        if (m.maxAge !== undefined) return t('feeModels.ageTo', { max: m.maxAge });
        return '';
    };
}

// ─── Assignment select (also used in the member dialog) ──────────────────────

type AssignmentSelectProps = {
    models: FeeModel[];
    value: string;
    onChange: (value: string) => void;
    disabled?: boolean;
    size?: 'small' | 'medium';
    label?: string;
};

/** Chooses how a member's fee model is picked: by age, a fixed model or none. */
export function FeeAssignmentSelect({ models, value, onChange, disabled, size, label }: AssignmentSelectProps) {
    const { t } = useTranslation();
    return (
        <FormControl fullWidth size={size} disabled={disabled}>
            {label && <InputLabel>{label}</InputLabel>}
            <Select value={value} label={label} onChange={(e) => onChange(e.target.value)}>
                <MenuItem value="auto">{t('feeModels.mode.auto')}</MenuItem>
                <MenuItem value="exempt">{t('feeModels.mode.exempt')}</MenuItem>
                {models.map((m) => <MenuItem key={m.id} value={m.id}>{m.name}</MenuItem>)}
            </Select>
        </FormControl>
    );
}

// ─── Model dialog ─────────────────────────────────────────────────────────────

type ModelDialogProps = {
    model: Partial<FeeModel>;
    onClose: () => void;
    onSaved: () => void;
};

function FeeModelDialog({ model, onClose, onSaved }: ModelDialogProps) {
    const { t } = useTranslation();
    const [form, setForm] = useState<Partial<FeeModel>>({ interval: 'yearly', proRata: false, ...model });
    const [error, setError] = useState<string | null>(null);
    const [saving, setSaving] = useState(false);

    const age = (key: 'minAge' | 'maxAge') => (e: React.ChangeEvent<HTMLInputElement>) =>
        setForm((prev) => ({ ...prev, [key]: e.target.value === '' ? undefined : Number(e.target.value) }));
    const ageInvalid = form.minAge !== undefined && form.maxAge !== undefined && form.minAge > form.maxAge;
    const valid = !!form.name?.trim() && (form.amount ?? 0) > 0 && !ageInvalid;

    const save = async () => {
        setSaving(true);
        setError(null);
        try {
            const data = { name: form.name!.trim(), amount: form.amount!, interval: form.interval!, minAge: form.minAge,
                maxAge: form.maxAge, proRata: !!form.proRata };
            if (form.id) await updateFeeModel({ ...data, id: form.id }); else await createFeeModel(data);
            onSaved();
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            setError(t(code === 'FEE_MODEL_NAME_TAKEN' ? 'feeModels.nameTaken' : 'feeModels.saveFailed'));
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{form.id ? t('feeModels.dialog.titleEdit') : t('feeModels.dialog.titleCreate')}</DialogTitle>
            <DialogContent>
                <Box display="grid" gap={2} mt={1} sx={{ gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
                    <TextField label={t('feeModels.fields.name')} value={form.name ?? ''} required autoFocus
                        onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))}
                        placeholder={t('feeModels.dialog.namePlaceholder')} sx={{ gridColumn: { sm: '1 / span 2' } }}
                        slotProps={{ htmlInput: { maxLength: 100 } }} />
                    <TextField label={t('feeModels.fields.amount')} type="number" value={form.amount ?? ''} required
                        onChange={(e) => setForm((prev) => ({ ...prev, amount: e.target.value === '' ? undefined : Number(e.target.value) }))}
                        slotProps={{ htmlInput: { min: 0, step: 0.01 } }} helperText={t('feeModels.dialog.amountHelp')} />
                    <FormControl fullWidth>
                        <InputLabel>{t('feeModels.fields.interval')}</InputLabel>
                        <Select value={form.interval} label={t('feeModels.fields.interval')}
                            onChange={(e) => setForm((prev) => ({ ...prev, interval: e.target.value as FeeModel['interval'] }))}>
                            {FEE_INTERVALS.map((i) => <MenuItem key={i} value={i}>{t(`feeModels.interval.${i}`)}</MenuItem>)}
                        </Select>
                    </FormControl>
                    <TextField label={t('feeModels.fields.minAge')} type="number" value={form.minAge ?? ''} onChange={age('minAge')}
                        slotProps={{ htmlInput: { min: 0, max: 150 } }} error={ageInvalid} />
                    <TextField label={t('feeModels.fields.maxAge')} type="number" value={form.maxAge ?? ''} onChange={age('maxAge')}
                        slotProps={{ htmlInput: { min: 0, max: 150 } }} error={ageInvalid}
                        helperText={ageInvalid ? t('feeModels.dialog.ageInvalid') : undefined} />
                    <Typography variant="body2" color="text.secondary" sx={{ gridColumn: { sm: '1 / span 2' } }}>
                        {t('feeModels.dialog.ageHelp')}
                    </Typography>
                    <FormControlLabel sx={{ gridColumn: { sm: '1 / span 2' } }}
                        control={<Checkbox checked={!!form.proRata} onChange={(e) => setForm((prev) => ({ ...prev, proRata: e.target.checked }))} />}
                        label={t('feeModels.dialog.proRata')} />
                </Box>
                {error && <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={save} disabled={!valid || saving}>{t('buttons.save')}</Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Models tab ───────────────────────────────────────────────────────────────

/** Fee models and the model every current member pays. */
export function FeeModelsTab({ canEdit }: { canEdit: boolean }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const ageLabel = useAgeLabel();
    const [models, setModels] = useState<FeeModel[] | null>(null);
    const [members, setMembers] = useState<MemberFeeModel[]>([]);
    const [editing, setEditing] = useState<Partial<FeeModel> | null>(null);
    const [deleting, setDeleting] = useState<FeeModel | null>(null);

    const reload = useCallback(async () => {
        try {
            const [m, a] = await Promise.all([fetchFeeModels(), fetchMemberFeeModels()]);
            setModels(m);
            setMembers(a);
        } catch {
            addNotification(t('feeModels.loadFailed'));
            setModels([]);
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const counts = useMemo(() => {
        const c = new Map<string, number>();
        for (const m of members) if (m.model) c.set(m.model.id, (c.get(m.model.id) ?? 0) + 1);
        return c;
    }, [members]);
    const withoutModel = members.filter((m) => !m.model && m.mode !== 'exempt');

    const assign = async (memberId: string, value: string) => {
        try {
            await saveFeeAssignment(memberId, toAssignment(value));
        } catch {
            addNotification(t('feeModels.saveFailed'));
        }
        await reload();
    };

    const modelColumns: GridColDef<FeeModel>[] = [
        { field: 'name', headerName: t('feeModels.fields.name'), flex: 1.5, minWidth: 150 },
        { field: 'amount', headerName: t('feeModels.fields.amount'), flex: 0.8, minWidth: 100, valueFormatter: (v: number) => money(v) },
        { field: 'interval', headerName: t('feeModels.fields.interval'), flex: 1, minWidth: 120, valueFormatter: (v: string) => t(`feeModels.interval.${v}`) },
        { field: 'age', headerName: t('feeModels.fields.age'), flex: 1, minWidth: 110, sortable: false, valueGetter: (_v, row) => ageLabel(row) },
        {
            field: 'proRata', headerName: t('feeModels.fields.proRata'), flex: 0.8, minWidth: 100,
            renderCell: ({ value }) => value ? <Chip size="small" label={t('feeModels.yes')} /> : '',
        },
        { field: 'members', headerName: t('feeModels.fields.members'), flex: 0.7, minWidth: 90, valueGetter: (_v, row) => counts.get(row.id) ?? 0 },
        ...(canEdit ? [{
            field: 'actions', headerName: t('members.table.header.actions'), width: 100, sortable: false,
            renderCell: ({ row }) => (
                <Box>
                    <Tooltip title={t('tooltips.edit')}>
                        <IconButton size="small" color="primary" onClick={() => setEditing(row)}><Edit fontSize="small" /></IconButton>
                    </Tooltip>
                    <Tooltip title={t('tooltips.delete')}>
                        <IconButton size="small" color="primary" onClick={() => setDeleting(row)}><Delete fontSize="small" /></IconButton>
                    </Tooltip>
                </Box>
            ),
        } as GridColDef<FeeModel>] : []),
    ];

    const memberColumns: GridColDef<MemberFeeModel>[] = [
        { field: 'member', headerName: t('finance.memberfees.fields.member'), flex: 1.5, minWidth: 180, valueGetter: (_v, row) => memberName(row.member) },
        { field: 'age', headerName: t('feeModels.columns.ageJan1'), flex: 0.6, minWidth: 90 },
        {
            field: 'assignment', headerName: t('feeModels.columns.assignment'), flex: 1.4, minWidth: 200, sortable: false,
            renderCell: ({ row }) => (
                <Box width="100%" display="flex" alignItems="center" height="100%">
                    <FeeAssignmentSelect models={models ?? []} value={assignmentValue(row)} size="small" disabled={!canEdit}
                        onChange={(v) => void assign(row.member.id, v)} />
                </Box>
            ),
        },
        {
            field: 'model', headerName: t('feeModels.columns.effective'), flex: 1.2, minWidth: 160,
            valueGetter: (_v, row) => row.model?.name ?? '',
            renderCell: ({ row }) => row.model
                ? `${row.model.name} · ${money(row.model.amount)} ${t(`feeModels.per.${row.model.interval}`)}`
                : <Chip size="small" color={row.mode === 'exempt' ? 'default' : 'warning'}
                    label={t(row.mode === 'exempt' ? 'feeModels.mode.exempt' : 'feeModels.noModel')} />,
        },
    ];

    if (models === null) return <CircularProgress />;

    return (
        <Box>
            {canEdit && (
                <Box display="flex" justifyContent="flex-end" mb={1}>
                    <Button variant="contained" startIcon={<Add />} onClick={() => setEditing({})}>{t('feeModels.create')}</Button>
                </Box>
            )}
            <DataGrid autoHeight rows={models} columns={modelColumns} hideFooter={models.length <= 10}
                localeText={{ noRowsLabel: t('feeModels.empty') }} />

            <Typography variant="h6" sx={{ mt: 4, mb: 1 }}>{t('feeModels.assignments.title')}</Typography>
            <Typography variant="body2" color="text.secondary" mb={1}>{t('feeModels.assignments.description')}</Typography>
            {withoutModel.length > 0 && (
                <Alert severity="warning" sx={{ mb: 1 }}>{t('feeModels.assignments.withoutModel', { count: withoutModel.length })}</Alert>
            )}
            <DataGrid autoHeight rows={members} getRowId={(r) => r.member.id} columns={memberColumns}
                pageSizeOptions={[10, 25, 50, 100]} initialState={{ pagination: { paginationModel: { pageSize: 25 } } }} />

            {editing && (
                <FeeModelDialog model={editing} onClose={() => setEditing(null)}
                    onSaved={() => { setEditing(null); void reload(); }} />
            )}
            <Dialog open={deleting !== null} onClose={() => setDeleting(null)}>
                <DialogTitle>{t('feeModels.deleteTitle')}</DialogTitle>
                <DialogContent>
                    <DialogContentText>{t('feeModels.deleteText', { name: deleting?.name })}</DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDeleting(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" color="error" onClick={async () => {
                        const m = deleting!;
                        setDeleting(null);
                        try {
                            await deleteFeeModel(m.id);
                        } catch {
                            addNotification(t('feeModels.saveFailed'));
                        }
                        void reload();
                    }}>{t('tooltips.delete')}</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}

// ─── Billing runs tab (Sollstellung) ──────────────────────────────────────────

const thisYear = new Date().getFullYear();

/** Creates the fees of a period from the fee models, with a preview first. */
export function FeeRunsTab({ canEdit, onFeesChanged }: { canEdit: boolean; onFeesChanged: () => void }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [from, setFrom] = useState(`${thisYear}-01`);
    const [to, setTo] = useState(`${thisYear}-12`);
    const [dueDate, setDueDate] = useState(`${thisYear}-03-01`);
    const [plan, setPlan] = useState<FeeRunPlan | null>(null);
    const [loading, setLoading] = useState(false);
    const [confirming, setConfirming] = useState(false);
    const [runs, setRuns] = useState<FeeRun[]>([]);
    const [deleting, setDeleting] = useState<FeeRun | null>(null);

    const reloadRuns = useCallback(async () => {
        try {
            setRuns(await fetchFeeRuns());
        } catch {
            addNotification(t('feeRuns.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reloadRuns(); }, [reloadRuns]);
    // A changed range invalidates the preview.
    useEffect(() => { setPlan(null); }, [from, to, dueDate]);

    const rangeValid = /^\d{4}-\d{2}$/.test(from) && /^\d{4}-\d{2}$/.test(to) && from <= to;

    const preview = async () => {
        setLoading(true);
        try {
            setPlan(await previewFeeRun({ from, to, dueDate: dueDate || undefined }));
        } catch {
            addNotification(t('feeRuns.previewFailed'));
        } finally {
            setLoading(false);
        }
    };

    const create = async () => {
        setConfirming(false);
        setLoading(true);
        try {
            const run = await createFeeRun({ from, to, dueDate: dueDate || undefined });
            addNotification(t('feeRuns.created', { count: run.count, total: money(run.total) }));
            setPlan(null);
            onFeesChanged();
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            addNotification(t(code === 'FEE_RUN_EMPTY' ? 'feeRuns.empty' : 'feeRuns.createFailed'));
        } finally {
            setLoading(false);
            void reloadRuns();
        }
    };

    const remove = async (run: FeeRun) => {
        setDeleting(null);
        try {
            const removed = await deleteFeeRun(run.id);
            addNotification(t('feeRuns.deleted', { count: removed }));
            onFeesChanged();
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            addNotification(t(code === 'FEE_RUN_IN_COLLECTION' ? 'feeRuns.inCollection' : 'feeRuns.deleteFailed'));
        }
        void reloadRuns();
    };

    const skippedByReason = useMemo(() => {
        const groups: Record<string, FeeRunPlan['skipped']> = {};
        for (const s of plan?.skipped ?? []) (groups[s.reason] ??= []).push(s);
        return groups;
    }, [plan]);

    const itemColumns: GridColDef<FeeRunPlan['items'][number]>[] = [
        { field: 'member', headerName: t('finance.memberfees.fields.member'), flex: 1.5, minWidth: 180, valueGetter: (_v, row) => memberName(row.member) },
        { field: 'description', headerName: t('finance.memberfees.fields.description'), flex: 1.3, minWidth: 160 },
        { field: 'amount', headerName: t('finance.memberfees.fields.amount'), flex: 0.8, minWidth: 100, valueFormatter: (v: number) => money(v) },
        {
            field: 'proRata', headerName: t('feeModels.fields.proRata'), flex: 0.8, minWidth: 110,
            renderCell: ({ row }) => row.proRata ? <Chip size="small" color="info" label={t('feeRuns.proRataOf', { full: money(row.model.amount) })} /> : '',
        },
        { field: 'dueDate', headerName: t('finance.memberfees.fields.dueDate'), flex: 0.8, minWidth: 110, valueFormatter: (v: string) => localDate(v) },
    ];

    const runColumns: GridColDef<FeeRun>[] = [
        { field: 'period', headerName: t('feeRuns.columns.period'), flex: 1, minWidth: 150, valueGetter: (_v, row) => `${row.from} – ${row.to}` },
        { field: 'count', headerName: t('feeRuns.columns.count'), flex: 0.6, minWidth: 90 },
        { field: 'total', headerName: t('feeRuns.columns.total'), flex: 0.8, minWidth: 110, valueFormatter: (v: number) => money(v) },
        { field: 'paidCount', headerName: t('feeRuns.columns.paid'), flex: 0.6, minWidth: 90, valueGetter: (_v, row) => `${row.paidCount} / ${row.count}` },
        { field: 'createdAt', headerName: t('feeRuns.columns.createdAt'), flex: 0.8, minWidth: 110, valueFormatter: (v: string) => new Date(v).toLocaleDateString() },
        ...(canEdit ? [{
            field: 'actions', headerName: t('members.table.header.actions'), width: 90, sortable: false,
            renderCell: ({ row }) => row.paidCount < row.count && (
                <Tooltip title={t('feeRuns.delete')}>
                    <IconButton size="small" color="primary" onClick={() => setDeleting(row)}><Delete fontSize="small" /></IconButton>
                </Tooltip>
            ),
        } as GridColDef<FeeRun>] : []),
    ];

    return (
        <Box>
            {canEdit && (
                <Paper sx={{ p: 2, mb: 3 }}>
                    <Typography variant="subtitle1" fontWeight={600} gutterBottom>{t('feeRuns.title')}</Typography>
                    <Typography variant="body2" color="text.secondary" mb={2}>{t('feeRuns.description')}</Typography>
                    <Box display="flex" gap={2} flexWrap="wrap" alignItems="flex-start">
                        <TextField label={t('feeRuns.from')} type="month" value={from} onChange={(e) => setFrom(e.target.value)}
                            slotProps={{ inputLabel: { shrink: true } }} error={!rangeValid} />
                        <TextField label={t('feeRuns.to')} type="month" value={to} onChange={(e) => setTo(e.target.value)}
                            slotProps={{ inputLabel: { shrink: true } }} error={!rangeValid}
                            helperText={rangeValid ? ' ' : t('feeRuns.rangeInvalid')} />
                        <TextField label={t('feeRuns.dueDate')} type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)}
                            slotProps={{ inputLabel: { shrink: true } }} helperText={t('feeRuns.dueDateHelp')} />
                        <Button variant="outlined" startIcon={loading ? <CircularProgress size={16} /> : <Preview />} sx={{ mt: 1 }}
                            onClick={preview} disabled={!rangeValid || loading}>
                            {t('feeRuns.preview')}
                        </Button>
                    </Box>

                    {plan && (
                        <Box mt={2}>
                            <Alert severity={plan.count > 0 ? 'info' : 'warning'} sx={{ mb: 2 }}
                                action={plan.count > 0 && (
                                    <Button color="inherit" variant="outlined" size="small" startIcon={<PlaylistAddCheck />}
                                        onClick={() => setConfirming(true)} disabled={loading}>
                                        {t('feeRuns.create')}
                                    </Button>
                                )}>
                                {plan.count > 0
                                    ? t('feeRuns.summary', { count: plan.count, total: money(plan.total) })
                                    : t('feeRuns.nothingToCharge')}
                            </Alert>
                            {plan.count > 0 && (
                                <DataGrid autoHeight rows={plan.items} columns={itemColumns}
                                    getRowId={(r) => `${r.member.id}-${r.period.start}`}
                                    pageSizeOptions={[10, 25, 50, 100]} initialState={{ pagination: { paginationModel: { pageSize: 10 } } }} />
                            )}
                            {(['ALREADY_CHARGED', 'NO_MODEL', 'EXEMPT'] as const).filter((r) => skippedByReason[r]).map((reason) => (
                                <Alert key={reason} severity={reason === 'NO_MODEL' ? 'warning' : 'success'} sx={{ mt: 2 }}>
                                    <Typography variant="subtitle2" gutterBottom>
                                        {t(`feeRuns.skipped.${reason}`, { count: skippedByReason[reason].length })}
                                    </Typography>
                                    <Typography variant="body2">
                                        {skippedByReason[reason].slice(0, 30).map((s) =>
                                            s.period ? `${memberName(s.member)} ${s.period.label}` : memberName(s.member)).join(', ')}
                                        {skippedByReason[reason].length > 30 && ' …'}
                                    </Typography>
                                </Alert>
                            ))}
                        </Box>
                    )}
                </Paper>
            )}

            <Typography variant="h6" sx={{ mb: 1 }}>{t('feeRuns.history')}</Typography>
            <DataGrid autoHeight rows={runs} columns={runColumns} pageSizeOptions={[10, 25]}
                localeText={{ noRowsLabel: t('feeRuns.noRuns') }} />

            <Dialog open={confirming} onClose={() => setConfirming(false)}>
                <DialogTitle>{t('feeRuns.confirmTitle')}</DialogTitle>
                <DialogContent>
                    <DialogContentText>
                        {plan && t('feeRuns.confirmText', { count: plan.count, total: money(plan.total), from, to })}
                    </DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setConfirming(false)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" onClick={create}>{t('feeRuns.create')}</Button>
                </DialogActions>
            </Dialog>
            <Dialog open={deleting !== null} onClose={() => setDeleting(null)}>
                <DialogTitle>{t('feeRuns.deleteTitle')}</DialogTitle>
                <DialogContent>
                    <DialogContentText>{t('feeRuns.deleteText', { count: deleting ? deleting.count - deleting.paidCount : 0 })}</DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDeleting(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" color="error" onClick={() => void remove(deleting!)}>{t('feeRuns.delete')}</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}

// ─── Open items ───────────────────────────────────────────────────────────────

type OpenItem = {
    id: string;
    member: string;
    count: number;
    total: number;
    oldestDue: string;
    overdue: boolean;
    fees: MemberFee[];
};

/** Unpaid fees per member and in total (Offene Posten). */
export function OpenItemsTab({ fees }: { fees: MemberFee[] }) {
    const { t } = useTranslation();
    const today = new Date().toISOString().slice(0, 10);
    const items = useMemo(() => {
        const byMember = new Map<string, OpenItem>();
        for (const f of fees) {
            if (f.paidDate) continue;
            const key = f.member?.id ?? f.memberId;
            const item = byMember.get(key) ?? {
                id: key, member: f.member ? memberName(f.member) : key, count: 0, total: 0, oldestDue: f.dueDate, overdue: false, fees: [],
            };
            item.count++;
            item.total += f.amount;
            item.fees.push(f);
            if (f.dueDate < item.oldestDue) item.oldestDue = f.dueDate;
            item.overdue = item.oldestDue.slice(0, 10) < today;
            byMember.set(key, item);
        }
        return [...byMember.values()].sort((a, b) => b.total - a.total);
    }, [fees, today]);
    const total = items.reduce((sum, i) => sum + i.total, 0);
    const overdue = fees.filter((f) => !f.paidDate && f.dueDate.slice(0, 10) < today).reduce((sum, f) => sum + f.amount, 0);

    const columns: GridColDef<OpenItem>[] = [
        { field: 'member', headerName: t('finance.memberfees.fields.member'), flex: 1.5, minWidth: 180 },
        { field: 'count', headerName: t('openItems.count'), flex: 0.6, minWidth: 90 },
        { field: 'total', headerName: t('openItems.total'), flex: 0.8, minWidth: 110, valueFormatter: (v: number) => money(v) },
        {
            field: 'oldestDue', headerName: t('openItems.oldestDue'), flex: 1.2, minWidth: 210,
            renderCell: ({ row }) => row.overdue
                ? <Chip size="small" color="error" label={t('openItems.overdueSince', { date: localDate(row.oldestDue) })} />
                : localDate(row.oldestDue),
        },
        {
            field: 'fees', headerName: t('openItems.fees'), flex: 1.6, minWidth: 200, sortable: false,
            valueGetter: (_v, row) => row.fees.map((f) => f.description ?? String(f.year)).join(', '),
        },
    ];

    return (
        <Box>
            <Box display="flex" gap={2} flexWrap="wrap" mb={2}>
                {[
                    { label: t('openItems.summary.total'), value: money(total), color: total > 0 ? 'warning.main' : 'success.main' },
                    { label: t('openItems.summary.overdue'), value: money(overdue), color: overdue > 0 ? 'error.main' : 'success.main' },
                    { label: t('openItems.summary.members'), value: String(items.length), color: 'text.primary' },
                ].map(({ label, value, color }) => (
                    <Box key={label} sx={{ flex: '1 1 150px', p: 2, borderRadius: 2, border: '1px solid', borderColor: 'divider', bgcolor: 'background.paper' }}>
                        <Typography variant="body2" color="text.secondary">{label}</Typography>
                        <Typography variant="h6" fontWeight={700} color={color}>{value}</Typography>
                    </Box>
                ))}
            </Box>
            <DataGrid autoHeight rows={items} columns={columns} pageSizeOptions={[10, 25, 50]}
                localeText={{ noRowsLabel: t('openItems.empty') }} />
        </Box>
    );
}
