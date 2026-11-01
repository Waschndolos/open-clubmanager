import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
    IconButton,
    Paper,
    Step,
    StepLabel,
    Stepper,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { AccountBalance, Cancel, Download, PlaylistAdd, TaskAlt } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { isAxiosError } from 'axios';
import {
    cancelSepaRun,
    confirmSepaRun,
    createSepaRun,
    downloadSepaRun,
    fetchSepaCandidates,
    fetchSepaCreditor,
    fetchSepaRuns,
    isValidIban,
    REMITTANCE_PLACEHOLDERS,
    saveSepaCreditor,
    SepaCandidate,
    SepaCreditor,
    SepaRun,
} from '../../api/sepa';
import { canEditFinance } from '../../api/types';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';

const money = (n: number) => `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €`;
const isoDate = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const localDate = (ymd: string) => new Date(`${ymd}T00:00:00`).toLocaleDateString();

/**
 * Whether the login may use direct debits: only roles that manage finances
 * (the treasurer), on a device that may access bank data and finances.
 */
export function useSepaAccess(): { visible: boolean; editable: boolean } {
    const { appRole } = useAuth();
    const bank = usePermission('membersBank');
    const finance = usePermission('finance');
    const visible = canEditFinance(appRole) && bank !== 'none' && finance !== 'none';
    return { visible, editable: visible && bank === 'write' && finance === 'write' };
}

// ─── Creditor settings ────────────────────────────────────────────────────────

/** Creditor identifier and club account, shown on the settings page. */
export function SepaCreditorSettings() {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const { visible, editable } = useSepaAccess();
    const [form, setForm] = useState<SepaCreditor>({ name: '', creditorId: '', iban: '' });
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (!visible) return;
        fetchSepaCreditor()
            .then((c) => c && setForm(c))
            .catch(() => addNotification(t('sepa.loadFailed')));
    }, [visible, addNotification, t]);

    if (!visible) return null;

    const ibanError = form.iban !== '' && !isValidIban(form.iban);
    const set = (key: keyof SepaCreditor) => (e: React.ChangeEvent<HTMLInputElement>) =>
        setForm((prev) => ({ ...prev, [key]: e.target.value }));

    const save = async () => {
        setSaving(true);
        try {
            setForm(await saveSepaCreditor(form));
            addNotification(t('sepa.settings.saved'));
        } catch (e) {
            const msg = isAxiosError(e) ? e.response?.data?.error : undefined;
            addNotification(t('sepa.settings.saveFailed', { reason: msg ?? '' }));
        } finally {
            setSaving(false);
        }
    };

    return (
        <>
            <Typography variant="h6" sx={{ mt: 4, mb: 1 }}>
                {t('sepa.settings.title')}
            </Typography>
            <Paper sx={{ p: 3 }}>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {t('sepa.settings.description')}
                </Typography>
                <Box component="fieldset" disabled={!editable} sx={{ border: 0, p: 0, m: 0, minWidth: 0, display: 'grid', gap: 2,
                    gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
                    <TextField label={t('sepa.settings.name')} value={form.name} onChange={set('name')} required
                        slotProps={{ htmlInput: { maxLength: 70 } }} />
                    <TextField label={t('sepa.settings.creditorId')} value={form.creditorId} onChange={set('creditorId')} required
                        helperText={t('sepa.settings.creditorIdHelp')} />
                    <TextField label={t('sepa.settings.iban')} value={form.iban} onChange={set('iban')} required
                        error={ibanError} helperText={ibanError ? t('sepa.settings.ibanInvalid') : ' '} />
                    <TextField label={t('sepa.settings.bic')} value={form.bic ?? ''} onChange={set('bic')}
                        helperText={t('sepa.settings.bicHelp')} />
                    <TextField label={t('sepa.settings.template')} value={form.remittanceTemplate ?? ''}
                        onChange={set('remittanceTemplate')} sx={{ gridColumn: { sm: '1 / span 2' } }}
                        placeholder="{description} Mitglied {number}"
                        helperText={t('sepa.settings.templateHelp', { placeholders: REMITTANCE_PLACEHOLDERS.join(' ') })} />
                </Box>
                {editable && (
                    <Button variant="contained" sx={{ mt: 2 }} onClick={save}
                        disabled={saving || !form.name || !form.creditorId || !form.iban || ibanError}>
                        {t('buttons.save')}
                    </Button>
                )}
            </Paper>
        </>
    );
}

// ─── Run wizard ───────────────────────────────────────────────────────────────

type WizardProps = {
    creditor: SepaCreditor;
    onClose: () => void;
    onCreated: (run: SepaRun) => void;
};

function SepaRunWizard({ creditor, onClose, onCreated }: WizardProps) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [step, setStep] = useState(0);
    const [candidates, setCandidates] = useState<SepaCandidate[] | null>(null);
    const [selected, setSelected] = useState<Set<string>>(new Set());
    const [collectionDate, setCollectionDate] = useState(() => {
        const d = new Date();
        d.setDate(d.getDate() + 5);
        return isoDate(d);
    });
    const [template, setTemplate] = useState(creditor.remittanceTemplate ?? '');
    const [creating, setCreating] = useState(false);
    const preselected = useRef(false);

    // Reloaded when the template changes, for the remittance preview.
    useEffect(() => {
        const timer = setTimeout(() => {
            fetchSepaCandidates(template)
                .then((list) => {
                    if (!preselected.current) {
                        preselected.current = true;
                        setSelected(new Set(list.filter((c) => c.issues.length === 0).map((c) => c.feeId)));
                    }
                    setCandidates(list);
                })
                .catch(() => addNotification(t('sepa.loadFailed')));
        }, 300);
        return () => clearTimeout(timer);
    }, [template, addNotification, t]);

    const valid = useMemo(() => (candidates ?? []).filter((c) => c.issues.length === 0), [candidates]);
    const invalid = useMemo(() => (candidates ?? []).filter((c) => c.issues.length > 0), [candidates]);
    const chosen = valid.filter((c) => selected.has(c.feeId));
    const total = chosen.reduce((sum, c) => sum + c.amount, 0);
    const tomorrow = (() => {
        const d = new Date();
        d.setDate(d.getDate() + 1);
        return isoDate(d);
    })();
    const dateValid = collectionDate >= tomorrow;

    const toggle = (id: string) => setSelected((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id); else next.add(id);
        return next;
    });

    const create = async () => {
        setCreating(true);
        try {
            const run = await createSepaRun({ collectionDate, remittanceTemplate: template, feeIds: chosen.map((c) => c.feeId) });
            await downloadSepaRun(run);
            onCreated(run);
        } catch (e) {
            const code = isAxiosError(e) ? e.response?.data?.code : undefined;
            addNotification(t(code === 'SEPA_FEES_INVALID' ? 'sepa.wizard.feesChanged' : 'sepa.wizard.createFailed'));
            setCreating(false);
        }
    };

    const steps = [t('sepa.wizard.steps.fees'), t('sepa.wizard.steps.details'), t('sepa.wizard.steps.summary')];

    return (
        <Dialog open onClose={onClose} maxWidth="lg" fullWidth>
            <DialogTitle>{t('sepa.wizard.title')}</DialogTitle>
            <DialogContent>
                <Stepper activeStep={step} sx={{ mb: 3, mt: 1 }}>
                    {steps.map((label) => <Step key={label}><StepLabel>{label}</StepLabel></Step>)}
                </Stepper>

                {candidates === null && <CircularProgress />}

                {candidates !== null && step === 0 && (
                    <Box>
                        {candidates.length === 0 && <Alert severity="info">{t('sepa.wizard.noOpenFees')}</Alert>}
                        {valid.length > 0 && (
                            <TableContainer sx={{ maxHeight: 360 }}>
                                <Table size="small" stickyHeader>
                                    <TableHead>
                                        <TableRow>
                                            <TableCell padding="checkbox">
                                                <Checkbox
                                                    checked={chosen.length === valid.length}
                                                    indeterminate={chosen.length > 0 && chosen.length < valid.length}
                                                    onChange={(e) => setSelected(new Set(e.target.checked ? valid.map((c) => c.feeId) : []))}
                                                    slotProps={{ input: { 'aria-label': t('sepa.wizard.selectAll') } }}
                                                />
                                            </TableCell>
                                            <TableCell>{t('sepa.columns.member')}</TableCell>
                                            <TableCell>{t('sepa.columns.fee')}</TableCell>
                                            <TableCell align="right">{t('sepa.columns.amount')}</TableCell>
                                            <TableCell>{t('sepa.columns.mandate')}</TableCell>
                                            <TableCell>{t('sepa.columns.iban')}</TableCell>
                                            <TableCell>{t('sepa.columns.sequence')}</TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {valid.map((c) => (
                                            <TableRow key={c.feeId} hover onClick={() => toggle(c.feeId)} sx={{ cursor: 'pointer' }}>
                                                <TableCell padding="checkbox"><Checkbox checked={selected.has(c.feeId)} /></TableCell>
                                                <TableCell>{c.member.firstName} {c.member.lastName} ({c.member.number})</TableCell>
                                                <TableCell>{c.description ?? c.year}</TableCell>
                                                <TableCell align="right">{money(c.amount)}</TableCell>
                                                <TableCell>{c.mandateReference}</TableCell>
                                                <TableCell sx={{ fontFamily: 'monospace' }}>{c.iban}</TableCell>
                                                <TableCell>
                                                    <Tooltip title={t(`sepa.sequence.${c.sequenceType}`)}>
                                                        <Chip size="small" label={c.sequenceType} variant="outlined" />
                                                    </Tooltip>
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </TableContainer>
                        )}
                        {invalid.length > 0 && (
                            <Alert severity="warning" sx={{ mt: 2 }}>
                                <Typography variant="subtitle2" gutterBottom>
                                    {t('sepa.wizard.excluded', { count: invalid.length })}
                                </Typography>
                                {invalid.map((c) => (
                                    <Box key={c.feeId} display="flex" alignItems="center" gap={1} flexWrap="wrap" mb={0.5}>
                                        <Typography variant="body2">
                                            {c.member.firstName} {c.member.lastName} ({c.member.number}) · {money(c.amount)}:
                                        </Typography>
                                        {c.issues.map((issue) => (
                                            <Chip key={issue} size="small" color="warning" label={t(`sepa.issues.${issue}`)} />
                                        ))}
                                    </Box>
                                ))}
                                <Typography variant="body2" mt={1}>{t('sepa.wizard.excludedHint')}</Typography>
                            </Alert>
                        )}
                    </Box>
                )}

                {candidates !== null && step === 1 && (
                    <Box display="flex" flexDirection="column" gap={2}>
                        <TextField
                            label={t('sepa.wizard.collectionDate')}
                            type="date"
                            value={collectionDate}
                            onChange={(e) => setCollectionDate(e.target.value)}
                            slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: tomorrow } }}
                            error={!dateValid}
                            helperText={dateValid ? t('sepa.wizard.collectionDateHelp') : t('sepa.wizard.collectionDateInvalid')}
                            sx={{ maxWidth: 260 }}
                        />
                        <TextField
                            label={t('sepa.settings.template')}
                            value={template}
                            onChange={(e) => setTemplate(e.target.value)}
                            placeholder="{description} Mitglied {number}"
                            helperText={t('sepa.settings.templateHelp', { placeholders: REMITTANCE_PLACEHOLDERS.join(' ') })}
                            fullWidth
                        />
                        <Typography variant="subtitle2">{t('sepa.wizard.preview')}</Typography>
                        {chosen.slice(0, 3).map((c) => (
                            <Typography key={c.feeId} variant="body2" sx={{ fontFamily: 'monospace' }}>
                                {c.member.lastName}: {c.remittance}
                            </Typography>
                        ))}
                    </Box>
                )}

                {candidates !== null && step === 2 && (
                    <Box display="flex" flexDirection="column" gap={1}>
                        <Typography>{t('sepa.wizard.summary', {
                            count: chosen.length, total: money(total), date: localDate(collectionDate),
                        })}</Typography>
                        <Typography variant="body2" color="text.secondary">
                            {t('sepa.wizard.summarySequence', {
                                first: chosen.filter((c) => c.sequenceType === 'FRST').length,
                                recurring: chosen.filter((c) => c.sequenceType === 'RCUR').length,
                            })}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                            {t('sepa.wizard.creditor', { name: creditor.name, creditorId: creditor.creditorId, iban: creditor.iban })}
                        </Typography>
                        <Alert severity="info" sx={{ mt: 1 }}>{t('sepa.wizard.afterExport')}</Alert>
                    </Box>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                {step > 0 && <Button onClick={() => setStep(step - 1)}>{t('sepa.wizard.back')}</Button>}
                {step < 2 && (
                    <Button variant="contained" onClick={() => setStep(step + 1)}
                        disabled={chosen.length === 0 || (step === 1 && !dateValid)}>
                        {t('sepa.wizard.next')}
                    </Button>
                )}
                {step === 2 && (
                    <Button variant="contained" startIcon={creating ? <CircularProgress size={16} /> : <Download />}
                        onClick={create} disabled={creating}>
                        {t('sepa.wizard.create')}
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
}

// ─── Runs tab ─────────────────────────────────────────────────────────────────

type TabProps = {
    /** Called after a run changed the state of fees. */
    onFeesChanged: () => void;
};

/** Direct debit runs on the finance page. */
export function SepaDirectDebitTab({ onFeesChanged }: TabProps) {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { addNotification } = useNotification();
    const { editable } = useSepaAccess();
    const [runs, setRuns] = useState<SepaRun[]>([]);
    const [creditor, setCreditor] = useState<SepaCreditor | null | undefined>(undefined);
    const [wizardOpen, setWizardOpen] = useState(false);
    const [confirming, setConfirming] = useState<{ run: SepaRun; date: string } | null>(null);
    const [cancelling, setCancelling] = useState<SepaRun | null>(null);

    const reload = useCallback(async () => {
        try {
            const [r, c] = await Promise.all([fetchSepaRuns(), fetchSepaCreditor()]);
            setRuns(r);
            setCreditor(c);
        } catch {
            addNotification(t('sepa.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const act = async (fn: () => Promise<unknown>) => {
        try {
            await fn();
            onFeesChanged();
        } catch {
            addNotification(t('sepa.runs.actionFailed'));
        }
        await reload();
    };

    const columns: GridColDef<SepaRun>[] = [
        {
            field: 'collectionDate', headerName: t('sepa.runs.collectionDate'), flex: 1, minWidth: 120,
            valueFormatter: (value: string) => localDate(value),
        },
        { field: 'count', headerName: t('sepa.runs.count'), flex: 0.6, minWidth: 90 },
        { field: 'total', headerName: t('sepa.runs.total'), flex: 1, minWidth: 110, valueFormatter: (value: number) => money(value) },
        {
            field: 'paidDate', headerName: t('sepa.runs.status'), flex: 1.2, minWidth: 160,
            renderCell: ({ row }) => row.paidDate
                ? <Chip size="small" color="success" label={t('sepa.runs.paid', { date: localDate(row.paidDate) })} />
                : <Chip size="small" color="info" label={t('sepa.runs.inCollection')} />,
        },
        {
            field: 'createdAt', headerName: t('sepa.runs.createdAt'), flex: 1, minWidth: 120,
            valueFormatter: (value: string) => new Date(value).toLocaleDateString(),
        },
        {
            field: 'actions', headerName: t('members.table.header.actions'), width: 140, sortable: false,
            renderCell: ({ row }) => (
                <Box>
                    <Tooltip title={t('sepa.runs.download')}>
                        <IconButton size="small" color="primary"
                            onClick={() => downloadSepaRun(row).catch(() => addNotification(t('sepa.runs.actionFailed')))}>
                            <Download fontSize="small" />
                        </IconButton>
                    </Tooltip>
                    {editable && !row.paidDate && (
                        <>
                            <Tooltip title={t('sepa.runs.confirm')}>
                                <IconButton size="small" color="success"
                                    onClick={() => setConfirming({ run: row, date: row.collectionDate })}>
                                    <TaskAlt fontSize="small" />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title={t('sepa.runs.cancel')}>
                                <IconButton size="small" color="error" onClick={() => setCancelling(row)}>
                                    <Cancel fontSize="small" />
                                </IconButton>
                            </Tooltip>
                        </>
                    )}
                </Box>
            ),
        },
    ];

    return (
        <Box>
            {creditor === null && (
                <Alert severity="info" sx={{ mb: 2 }}
                    action={<Button color="inherit" size="small" onClick={() => navigate('/settings')}>{t('sepa.runs.openSettings')}</Button>}>
                    {t('sepa.runs.notConfigured')}
                </Alert>
            )}
            {editable && (
                <Box display="flex" justifyContent="flex-end" mb={1}>
                    <Button variant="contained" startIcon={<PlaylistAdd />} disabled={!creditor} onClick={() => setWizardOpen(true)}>
                        {t('sepa.runs.create')}
                    </Button>
                </Box>
            )}
            <DataGrid
                autoHeight
                rows={runs}
                columns={columns}
                pageSizeOptions={[10, 25, 50]}
                localeText={{ noRowsLabel: t('sepa.runs.empty') }}
            />

            {wizardOpen && creditor && (
                <SepaRunWizard
                    creditor={creditor}
                    onClose={() => setWizardOpen(false)}
                    onCreated={() => {
                        setWizardOpen(false);
                        addNotification(t('sepa.wizard.created'));
                        onFeesChanged();
                        void reload();
                    }}
                />
            )}

            <Dialog open={confirming !== null} onClose={() => setConfirming(null)}>
                <DialogTitle>{t('sepa.runs.confirmTitle')}</DialogTitle>
                <DialogContent>
                    <DialogContentText sx={{ mb: 2 }}>{t('sepa.runs.confirmText')}</DialogContentText>
                    <TextField
                        label={t('sepa.runs.paidDate')}
                        type="date"
                        value={confirming?.date ?? ''}
                        onChange={(e) => setConfirming((prev) => prev && { ...prev, date: e.target.value })}
                        slotProps={{ inputLabel: { shrink: true } }}
                    />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setConfirming(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" color="success" startIcon={<AccountBalance />} disabled={!confirming?.date}
                        onClick={() => {
                            const c = confirming!;
                            setConfirming(null);
                            void act(() => confirmSepaRun(c.run.id, c.date));
                        }}>
                        {t('sepa.runs.confirm')}
                    </Button>
                </DialogActions>
            </Dialog>

            <Dialog open={cancelling !== null} onClose={() => setCancelling(null)}>
                <DialogTitle>{t('sepa.runs.cancelTitle')}</DialogTitle>
                <DialogContent>
                    <DialogContentText>{t('sepa.runs.cancelText')}</DialogContentText>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setCancelling(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" color="error"
                        onClick={() => {
                            const run = cancelling!;
                            setCancelling(null);
                            void act(() => cancelSepaRun(run.id));
                        }}>
                        {t('sepa.runs.cancel')}
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}
