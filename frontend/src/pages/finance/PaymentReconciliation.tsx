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
    DialogTitle,
    IconButton,
    InputAdornment,
    Paper,
    Stack,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    TextField,
    ToggleButton,
    ToggleButtonGroup,
    Tooltip,
    Typography,
} from '@mui/material';
import { Block, Check, DoneAll, Edit, Search, Undo } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { isAxiosError } from 'axios';
import {
    Confidence,
    confirmCertainReconciliations,
    confirmReconciliation,
    dismissReconciliation,
    FeeAllocation,
    fetchReconciliation,
    Reconciliation,
    ReconciliationEntry,
    ReconciliationFee,
    reopenReconciliation,
} from '../../api/reconciliation';
import { useNotification } from '../../context/NotificationContext';

const money = (n: number) => `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €`;
const localDate = (ymd: string) => new Date(`${ymd}T00:00:00`).toLocaleDateString();
const cents = (n: number) => Math.round(n * 100);

const CONFIDENCE_COLOR: Record<Confidence, 'success' | 'info' | 'warning'> = {
    certain: 'success',
    probable: 'info',
    unclear: 'warning',
};

type Filter = 'open' | 'done' | 'all';

function feeLabel(f: { member: { firstName: string; lastName: string; number: number }; description?: string; year: number }) {
    return `${f.member.firstName} ${f.member.lastName} (${f.member.number}) · ${f.description ?? f.year}`;
}

/** What a fee can take: the rest due for a payment, what was paid for a return. */
function capacity(fee: ReconciliationFee, isReturn: boolean): number {
    return isReturn ? Math.min(fee.paid, fee.amount) : fee.amount - fee.paid;
}

// ─── Assignment dialog ────────────────────────────────────────────────────────

type AssignDialogProps = {
    entry: ReconciliationEntry;
    fees: ReconciliationFee[];
    onClose: () => void;
    onConfirm: (allocations: FeeAllocation[]) => Promise<void>;
};

/** Lets the treasurer correct a suggestion or assign a transaction by hand. */
function AssignDialog({ entry, fees, onClose, onConfirm }: AssignDialogProps) {
    const { t } = useTranslation();
    const [amounts, setAmounts] = useState<Map<string, string>>(
        () => new Map((entry.suggestion?.allocations ?? []).map((a) => [a.feeId, a.amount.toFixed(2)]))
    );
    const [query, setQuery] = useState('');
    const [saving, setSaving] = useState(false);

    // Payments go to open fees, returns reverse paid ones.
    const assignable = useMemo(
        () => fees.filter((f) => (entry.isReturn ? f.paid > 0 : f.open) && capacity(f, entry.isReturn) > 0),
        [fees, entry.isReturn]
    );
    const visible = assignable.filter((f) => amounts.has(f.feeId)
        || feeLabel(f).toLowerCase().includes(query.trim().toLowerCase()));
    const parsed = [...amounts].map(([feeId, value]) => ({ feeId, amount: Number(value.replace(',', '.')) }));
    const total = parsed.reduce((sum, a) => sum + (Number.isFinite(a.amount) ? a.amount : 0), 0);
    const byId = new Map(assignable.map((f) => [f.feeId, f]));
    const invalid = parsed.filter((a) => {
        const fee = byId.get(a.feeId);
        return !fee || !Number.isFinite(a.amount) || a.amount <= 0 || cents(a.amount) > cents(capacity(fee, entry.isReturn));
    });
    const exceeds = cents(total) > cents(entry.amount);

    const toggle = (fee: ReconciliationFee) => setAmounts((prev) => {
        const next = new Map(prev);
        if (next.has(fee.feeId)) {
            next.delete(fee.feeId);
        } else {
            const left = Math.max(0, entry.amount - total);
            next.set(fee.feeId, Math.min(left, capacity(fee, entry.isReturn)).toFixed(2));
        }
        return next;
    });

    const confirm = async () => {
        setSaving(true);
        try {
            await onConfirm(parsed);
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>{t(entry.isReturn ? 'reconciliation.dialog.titleReturn' : 'reconciliation.dialog.title')}</DialogTitle>
            <DialogContent>
                <Paper variant="outlined" sx={{ p: 2, mb: 2 }}>
                    <Typography variant="subtitle2">
                        {localDate(entry.date)} · {entry.counterpartyName ?? t('reconciliation.unknownPayer')} · {money(entry.amount)}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">{entry.description}</Typography>
                </Paper>
                <TextField
                    size="small"
                    fullWidth
                    placeholder={t('reconciliation.dialog.search')}
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    slotProps={{ input: { startAdornment: <InputAdornment position="start"><Search fontSize="small" /></InputAdornment> } }}
                    sx={{ mb: 1 }}
                />
                {assignable.length === 0 ? (
                    <Alert severity="info">{t(entry.isReturn ? 'reconciliation.dialog.noPaidFees' : 'reconciliation.dialog.noOpenFees')}</Alert>
                ) : (
                    <TableContainer sx={{ maxHeight: 340 }}>
                        <Table size="small" stickyHeader>
                            <TableHead>
                                <TableRow>
                                    <TableCell padding="checkbox" />
                                    <TableCell>{t('reconciliation.columns.fee')}</TableCell>
                                    <TableCell align="right">
                                        {t(entry.isReturn ? 'reconciliation.dialog.paid' : 'reconciliation.dialog.due')}
                                    </TableCell>
                                    <TableCell align="right" sx={{ width: 160 }}>{t('reconciliation.dialog.assigned')}</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {visible.map((f) => {
                                    const value = amounts.get(f.feeId);
                                    const bad = value !== undefined && invalid.some((a) => a.feeId === f.feeId);
                                    return (
                                        <TableRow key={f.feeId} hover selected={value !== undefined}>
                                            <TableCell padding="checkbox">
                                                <Checkbox checked={value !== undefined} onChange={() => toggle(f)}
                                                    slotProps={{ input: { 'aria-label': feeLabel(f) } }} />
                                            </TableCell>
                                            <TableCell>{feeLabel(f)}</TableCell>
                                            <TableCell align="right">{money(capacity(f, entry.isReturn))}</TableCell>
                                            <TableCell align="right">
                                                {value !== undefined && (
                                                    <TextField
                                                        size="small"
                                                        value={value}
                                                        error={bad}
                                                        onChange={(e) => setAmounts((prev) => new Map(prev).set(f.feeId, e.target.value))}
                                                        slotProps={{
                                                            htmlInput: { inputMode: 'decimal', 'aria-label': t('reconciliation.dialog.assigned') },
                                                            input: { endAdornment: <InputAdornment position="end">€</InputAdornment> },
                                                        }}
                                                    />
                                                )}
                                            </TableCell>
                                        </TableRow>
                                    );
                                })}
                            </TableBody>
                        </Table>
                    </TableContainer>
                )}
                <Box display="flex" justifyContent="space-between" mt={2} gap={2} flexWrap="wrap">
                    <Typography variant="body2" color={exceeds ? 'error' : 'text.secondary'}>
                        {t('reconciliation.dialog.sum', { assigned: money(total), amount: money(entry.amount) })}
                    </Typography>
                    {!exceeds && cents(total) < cents(entry.amount) && parsed.length > 0 && (
                        <Typography variant="body2" color="text.secondary">
                            {t('reconciliation.dialog.rest', { rest: money(entry.amount - total) })}
                        </Typography>
                    )}
                </Box>
                {invalid.length > 0 && <Alert severity="error" sx={{ mt: 1 }}>{t('reconciliation.dialog.invalidAmount')}</Alert>}
                {exceeds && <Alert severity="error" sx={{ mt: 1 }}>{t('reconciliation.dialog.exceeds')}</Alert>}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button
                    variant="contained"
                    startIcon={saving ? <CircularProgress size={16} /> : <Check />}
                    disabled={saving || parsed.length === 0 || invalid.length > 0 || exceeds}
                    onClick={confirm}
                >
                    {t('reconciliation.actions.confirm')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Reconciliation tab ───────────────────────────────────────────────────────

type TabProps = {
    editable: boolean;
    /** Called after fees were paid or opened again. */
    onFeesChanged: () => void;
};

/** Review of imported bank transactions against open member fees. */
export function PaymentReconciliationTab({ editable, onFeesChanged }: TabProps) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [data, setData] = useState<Reconciliation | null>(null);
    const [filter, setFilter] = useState<Filter>('open');
    const [editing, setEditing] = useState<ReconciliationEntry | null>(null);
    const [busy, setBusy] = useState(false);

    const reload = useCallback(async () => {
        try {
            setData(await fetchReconciliation());
        } catch {
            addNotification(t('reconciliation.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const fees = useMemo(() => new Map((data?.fees ?? []).map((f) => [f.feeId, f])), [data]);
    const entries = (data?.entries ?? []).filter((e) => e.status !== 'none');
    const open = entries.filter((e) => e.status === 'open');
    const certain = open.filter((e) => e.suggestion?.confidence === 'certain');
    const shown = entries.filter((e) => filter === 'all' || (filter === 'open') === (e.status === 'open'));

    const run = async (fn: () => Promise<unknown>, success?: string) => {
        setBusy(true);
        try {
            await fn();
            if (success) addNotification(success);
            onFeesChanged();
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            addNotification(t(code === 'FEES_NOT_ASSIGNABLE' ? 'reconciliation.feesChanged' : 'reconciliation.actionFailed'));
        } finally {
            setBusy(false);
        }
        await reload();
    };

    const confirmAllCertain = () => run(async () => {
        const count = await confirmCertainReconciliations();
        addNotification(t('reconciliation.certainConfirmed', { count }));
    });

    if (data === null) return <CircularProgress />;

    const counts = (c: Confidence) => open.filter((e) => e.suggestion?.confidence === c).length;

    return (
        <Box>
            <Box display="flex" alignItems="center" justifyContent="space-between" gap={2} flexWrap="wrap" mb={2}>
                <Box display="flex" alignItems="center" gap={1} flexWrap="wrap">
                    <ToggleButtonGroup size="small" exclusive value={filter} onChange={(_, v) => v && setFilter(v)}>
                        <ToggleButton value="open">{t('reconciliation.filter.open', { count: open.length })}</ToggleButton>
                        <ToggleButton value="done">{t('reconciliation.filter.done')}</ToggleButton>
                        <ToggleButton value="all">{t('reconciliation.filter.all')}</ToggleButton>
                    </ToggleButtonGroup>
                    {(['certain', 'probable', 'unclear'] as Confidence[]).map((c) => counts(c) > 0 && (
                        <Chip key={c} size="small" variant="outlined" color={CONFIDENCE_COLOR[c]}
                            label={`${t(`reconciliation.confidence.${c}`)}: ${counts(c)}`} />
                    ))}
                </Box>
                {editable && (
                    <Button variant="contained" startIcon={<DoneAll />} disabled={busy || certain.length === 0} onClick={confirmAllCertain}>
                        {t('reconciliation.actions.confirmCertain', { count: certain.length })}
                    </Button>
                )}
            </Box>

            {entries.length === 0 && <Alert severity="info">{t('reconciliation.empty')}</Alert>}
            {entries.length > 0 && shown.length === 0 && <Alert severity="success">{t('reconciliation.allDone')}</Alert>}

            {shown.length > 0 && (
                <TableContainer component={Paper} variant="outlined">
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('reconciliation.columns.date')}</TableCell>
                                <TableCell>{t('reconciliation.columns.payer')}</TableCell>
                                <TableCell>{t('reconciliation.columns.remittance')}</TableCell>
                                <TableCell align="right">{t('reconciliation.columns.amount')}</TableCell>
                                <TableCell>{t('reconciliation.columns.assignment')}</TableCell>
                                {editable && <TableCell align="right">{t('members.table.header.actions')}</TableCell>}
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {shown.map((e) => {
                                const allocations = e.status === 'open'
                                    ? (e.suggestion?.allocations ?? []).map((a) => ({ ...a, fee: fees.get(a.feeId) }))
                                    : e.payments.map((p) => ({ feeId: p.feeId, amount: p.amount, fee: p }));
                                return (
                                    <TableRow key={e.transactionId} hover>
                                        <TableCell sx={{ whiteSpace: 'nowrap' }}>{localDate(e.date)}</TableCell>
                                        <TableCell>
                                            <Typography variant="body2">{e.counterpartyName ?? t('reconciliation.unknownPayer')}</Typography>
                                            {e.counterpartyIban && (
                                                <Typography variant="caption" color="text.secondary" sx={{ fontFamily: 'monospace' }}>
                                                    {e.counterpartyIban}
                                                </Typography>
                                            )}
                                        </TableCell>
                                        <TableCell sx={{ maxWidth: 220 }}>
                                            <Typography variant="body2">{e.description}</Typography>
                                            {e.isReturn && (
                                                <Chip size="small" color="error" variant="outlined"
                                                    sx={{ mt: 0.5, height: 'auto', py: 0.25, '& .MuiChip-label': { whiteSpace: 'normal' } }}
                                                    label={e.returnReason
                                                        ? `${t('reconciliation.return')} · ${t(`reconciliation.returnReasons.${e.returnReason}`, { defaultValue: e.returnReason })}`
                                                        : t('reconciliation.return')} />
                                            )}
                                        </TableCell>
                                        <TableCell align="right" sx={{ whiteSpace: 'nowrap', color: e.isReturn ? 'error.main' : undefined }}>
                                            {e.isReturn ? '−' : ''}{money(e.amount)}
                                        </TableCell>
                                        <TableCell sx={{ minWidth: 220 }}>
                                            <Box mb={0.5}>
                                                {e.status === 'open' && e.suggestion && (
                                                    <Chip size="small" color={CONFIDENCE_COLOR[e.suggestion.confidence]}
                                                        label={t(`reconciliation.confidence.${e.suggestion.confidence}`)} />
                                                )}
                                                {e.status === 'open' && !e.suggestion && (
                                                    <Chip size="small" label={t('reconciliation.status.unmatched')} />
                                                )}
                                                {e.status !== 'open' && (
                                                    <Chip size="small" variant="outlined"
                                                        color={e.status === 'reconciled' ? 'success' : 'default'}
                                                        label={t(`reconciliation.status.${e.status}`)} />
                                                )}
                                            </Box>
                                            {allocations.length === 0 && (
                                                <Typography variant="body2" color="text.secondary">
                                                    {e.status === 'dismissed' ? t('reconciliation.noFee') : t('reconciliation.noSuggestion')}
                                                </Typography>
                                            )}
                                            <Stack spacing={0.5}>
                                                {allocations.map((a) => (
                                                    <Box key={a.feeId}>
                                                        <Typography variant="body2">
                                                            {a.fee ? `${a.fee.member.firstName} ${a.fee.member.lastName} (${a.fee.member.number})` : a.feeId}
                                                        </Typography>
                                                        <Typography variant="caption" color="text.secondary">
                                                            {a.fee ? `${a.fee.description ?? a.fee.year} · ` : ''}{money(a.amount)}
                                                        </Typography>
                                                    </Box>
                                                ))}
                                            </Stack>
                                            {e.status === 'open' && e.suggestion && (
                                                <Box display="flex" gap={0.5} flexWrap="wrap" mt={0.5}>
                                                    {e.suggestion.reasons.map((r) => (
                                                        <Chip key={r} size="small" variant="outlined" label={t(`reconciliation.reasons.${r}`)} />
                                                    ))}
                                                </Box>
                                            )}
                                            {e.status === 'open' && e.suggestion && e.suggestion.unallocated > 0 && (
                                                <Typography variant="caption" color="text.secondary">
                                                    {t('reconciliation.unallocated', { amount: money(e.suggestion.unallocated) })}
                                                </Typography>
                                            )}
                                        </TableCell>
                                        {editable && (
                                            <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                                                {e.status === 'open' ? (
                                                    <>
                                                        {e.suggestion && (
                                                            <Tooltip title={t('reconciliation.actions.confirm')}>
                                                                <span>
                                                                    <IconButton size="small" color="success" disabled={busy}
                                                                        onClick={() => run(() => confirmReconciliation(e.transactionId, e.suggestion!.allocations),
                                                                            t('reconciliation.confirmed'))}>
                                                                        <Check fontSize="small" />
                                                                    </IconButton>
                                                                </span>
                                                            </Tooltip>
                                                        )}
                                                        <Tooltip title={t('reconciliation.actions.edit')}>
                                                            <IconButton size="small" color="primary" disabled={busy} onClick={() => setEditing(e)}>
                                                                <Edit fontSize="small" />
                                                            </IconButton>
                                                        </Tooltip>
                                                        <Tooltip title={t('reconciliation.actions.dismiss')}>
                                                            <IconButton size="small" disabled={busy}
                                                                onClick={() => run(() => dismissReconciliation(e.transactionId))}>
                                                                <Block fontSize="small" />
                                                            </IconButton>
                                                        </Tooltip>
                                                    </>
                                                ) : (
                                                    <Tooltip title={t('reconciliation.actions.undo')}>
                                                        <IconButton size="small" disabled={busy}
                                                            onClick={() => run(() => reopenReconciliation(e.transactionId), t('reconciliation.reopened'))}>
                                                            <Undo fontSize="small" />
                                                        </IconButton>
                                                    </Tooltip>
                                                )}
                                            </TableCell>
                                        )}
                                    </TableRow>
                                );
                            })}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}

            <Typography variant="body2" color="text.secondary" mt={1}>{t('reconciliation.hint')}</Typography>

            {editing && (
                <AssignDialog
                    entry={editing}
                    fees={data.fees}
                    onClose={() => setEditing(null)}
                    onConfirm={async (allocations) => {
                        const entry = editing;
                        setEditing(null);
                        await run(() => confirmReconciliation(entry.transactionId, allocations), t('reconciliation.confirmed'));
                    }}
                />
            )}
        </Box>
    );
}
