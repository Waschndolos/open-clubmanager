import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Alert,
    Autocomplete,
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
    FormControlLabel,
    IconButton,
    MenuItem,
    Paper,
    Radio,
    RadioGroup,
    Stack,
    Switch,
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
import { Block, ContentCopy, Delete, Download, Edit, LinkOff, PersonAdd, ReceiptLong, VolunteerActivism } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { isAxiosError } from 'axios';
import {
    assignDonation,
    createAnnualReceipts,
    createDonationReceipt,
    createDonor,
    deleteDonor,
    Donation,
    DonationCandidate,
    DonationReceipt,
    DonationSettings,
    Donor,
    downloadDonationReceipts,
    duplicateDonationReceipt,
    fetchDonationCandidates,
    fetchDonationReceipts,
    fetchDonations,
    fetchDonationSettings,
    fetchDonors,
    saveDonationSettings,
    unassignDonation,
    updateDonor,
    voidDonationReceipt,
} from '../../api/donations';
import { fetchMembers } from '../../api/members';
import { canEditFinance, Member } from '../../api/types';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';

const money = (n: number) => `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} €`;
const localDate = (ymd: string) => new Date(`${ymd.slice(0, 10)}T00:00:00`).toLocaleDateString();
const isoDate = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

/**
 * Whether the login may use donation receipts: only roles that manage
 * finances (the treasurer), on a device that may access the finances and
 * the members' addresses.
 */
export function useDonationAccess(): { visible: boolean; editable: boolean } {
    const { appRole } = useAuth();
    const finance = usePermission('finance');
    const contact = usePermission('membersContact');
    const visible = canEditFinance(appRole) && finance !== 'none' && contact !== 'none';
    return { visible, editable: visible && finance === 'write' };
}

function errorCode(e: unknown): string | undefined {
    return isAxiosError(e) ? e.response?.data?.code : undefined;
}

// ─── Settings ─────────────────────────────────────────────────────────────────

const EMPTY_SETTINGS: DonationSettings = {
    issuerName: '', issuerAddress: '', taxOffice: '', taxNumber: '', exemptionKind: 'notice', exemptionDate: '',
    purposes: '', feesDeductible: false,
};

/** The club's exemption data for receipts, shown on the settings page. */
export function DonationSettingsCard() {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const { visible, editable } = useDonationAccess();
    const [form, setForm] = useState<DonationSettings>(EMPTY_SETTINGS);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        if (!visible) return;
        fetchDonationSettings()
            .then((s) => s && setForm(s))
            .catch(() => addNotification(t('donations.loadFailed')));
    }, [visible, addNotification, t]);

    if (!visible) return null;

    const set = (key: keyof DonationSettings) => (e: React.ChangeEvent<HTMLInputElement>) =>
        setForm((prev) => ({ ...prev, [key]: e.target.value }));
    const complete = form.issuerName && form.issuerAddress && form.taxOffice && form.taxNumber && form.exemptionDate && form.purposes;

    const save = async () => {
        setSaving(true);
        try {
            setForm(await saveDonationSettings(form));
            addNotification(t('donations.settings.saved'));
        } catch (e) {
            const msg = isAxiosError(e) ? e.response?.data?.error : undefined;
            addNotification(t('donations.settings.saveFailed', { reason: msg ?? '' }));
        } finally {
            setSaving(false);
        }
    };

    return (
        <>
            <Typography variant="h6" sx={{ mt: 4, mb: 1 }}>
                {t('donations.settings.title')}
            </Typography>
            <Paper sx={{ p: 3 }}>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {t('donations.settings.description')}
                </Typography>
                {form.exemptionExpired && (
                    <Alert severity="warning" sx={{ mb: 2 }}>
                        {t('donations.exemptionExpired', { date: localDate(form.validUntil ?? form.exemptionDate) })}
                    </Alert>
                )}
                <Box component="fieldset" disabled={!editable} sx={{ border: 0, p: 0, m: 0, minWidth: 0, display: 'grid', gap: 2,
                    gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' } }}>
                    <TextField label={t('donations.settings.issuerName')} value={form.issuerName} onChange={set('issuerName')} required />
                    <TextField label={t('donations.settings.issuerAddress')} value={form.issuerAddress} onChange={set('issuerAddress')}
                        required multiline minRows={2} />
                    <TextField label={t('donations.settings.taxOffice')} value={form.taxOffice} onChange={set('taxOffice')} required />
                    <TextField label={t('donations.settings.taxNumber')} value={form.taxNumber} onChange={set('taxNumber')} required />
                    <Box sx={{ gridColumn: { sm: '1 / span 2' } }}>
                        <RadioGroup row value={form.exemptionKind}
                            onChange={(e) => setForm((prev) => ({ ...prev, exemptionKind: e.target.value as DonationSettings['exemptionKind'] }))}>
                            <FormControlLabel value="notice" control={<Radio />} label={t('donations.settings.kind.notice')} />
                            <FormControlLabel value="assessment" control={<Radio />} label={t('donations.settings.kind.assessment')} />
                        </RadioGroup>
                    </Box>
                    <TextField label={t('donations.settings.exemptionDate')} type="date" value={form.exemptionDate}
                        onChange={set('exemptionDate')} required slotProps={{ inputLabel: { shrink: true } }}
                        helperText={t('donations.settings.exemptionDateHelp')} />
                    {form.exemptionKind === 'notice' ? (
                        <TextField label={t('donations.settings.assessmentPeriod')} value={form.assessmentPeriod ?? ''}
                            onChange={set('assessmentPeriod')} placeholder="2024" />
                    ) : <Box />}
                    <TextField label={t('donations.settings.purposes')} value={form.purposes} onChange={set('purposes')} required
                        sx={{ gridColumn: { sm: '1 / span 2' } }} placeholder={t('donations.settings.purposesPlaceholder')}
                        helperText={t('donations.settings.purposesHelp')} />
                    <TextField label={t('donations.settings.place')} value={form.place ?? ''} onChange={set('place')} />
                    <TextField label={t('donations.settings.signatory')} value={form.signatory ?? ''} onChange={set('signatory')} />
                    <FormControlLabel sx={{ gridColumn: { sm: '1 / span 2' } }}
                        control={<Switch checked={form.feesDeductible}
                            onChange={(e) => setForm((prev) => ({ ...prev, feesDeductible: e.target.checked }))} />}
                        label={t('donations.settings.feesDeductible')} />
                </Box>
                {editable && (
                    <Button variant="contained" sx={{ mt: 2 }} onClick={save} disabled={saving || !complete}>
                        {t('buttons.save')}
                    </Button>
                )}
            </Paper>
        </>
    );
}

// ─── Donor dialog ─────────────────────────────────────────────────────────────

function DonorDialog({ donor, onClose, onSaved }: { donor: Partial<Donor>; onClose: () => void; onSaved: (d: Donor) => void }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [form, setForm] = useState<Partial<Donor>>(donor);
    const set = (key: keyof Donor) => (e: React.ChangeEvent<HTMLInputElement>) => setForm((prev) => ({ ...prev, [key]: e.target.value }));

    const save = async () => {
        const data = { name: form.name ?? '', street: form.street, postalCode: form.postalCode, city: form.city, notes: form.notes };
        try {
            onSaved(form.id ? await updateDonor(form.id, data) : await createDonor(data));
        } catch {
            addNotification(t('donations.actionFailed'));
        }
    };

    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{form.id ? t('donations.donors.edit') : t('donations.donors.create')}</DialogTitle>
            <DialogContent>
                <Box display="grid" gap={2} mt={1} sx={{ gridTemplateColumns: '1fr 2fr' }}>
                    <TextField label={t('donations.donors.name')} value={form.name ?? ''} onChange={set('name')} required
                        sx={{ gridColumn: '1 / span 2' }} />
                    <TextField label={t('donations.donors.street')} value={form.street ?? ''} onChange={set('street')}
                        sx={{ gridColumn: '1 / span 2' }} />
                    <TextField label={t('donations.donors.postalCode')} value={form.postalCode ?? ''} onChange={set('postalCode')} />
                    <TextField label={t('donations.donors.city')} value={form.city ?? ''} onChange={set('city')} />
                    <TextField label={t('donations.donors.notes')} value={form.notes ?? ''} onChange={set('notes')} multiline
                        sx={{ gridColumn: '1 / span 2' }} />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={save} disabled={!form.name?.trim()}>{t('buttons.save')}</Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Assign dialog ────────────────────────────────────────────────────────────

type Party = { kind: 'member'; member: Member } | { kind: 'donor'; donor: Donor };

const partyLabel = (p: Party) => p.kind === 'member'
    ? `${p.member.firstName} ${p.member.lastName} (${p.member.number})`
    : p.donor.name;

type AssignProps = {
    donation: Donation;
    members: Member[];
    donors: Donor[];
    onClose: () => void;
    onSaved: () => void;
    onNewDonor: () => void;
};

function AssignDialog({ donation, members, donors, onClose, onSaved, onNewDonor }: AssignProps) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const options = useMemo<Party[]>(() => [
        ...members.map((m) => ({ kind: 'member' as const, member: m })),
        ...donors.map((d) => ({ kind: 'donor' as const, donor: d })),
    ], [members, donors]);
    const [party, setParty] = useState<Party | null>(() =>
        options.find((o) => (o.kind === 'member' && o.member.id === donation.member?.id) ||
            (o.kind === 'donor' && o.donor.id === donation.donor?.id)) ?? null);
    const [waiver, setWaiver] = useState(donation.expenseWaiver);

    const save = async () => {
        if (!party) return;
        try {
            await assignDonation(donation.transactionId, party.kind === 'member'
                ? { memberId: party.member.id, expenseWaiver: waiver }
                : { donorId: party.donor.id, expenseWaiver: waiver });
            onSaved();
        } catch (e) {
            addNotification(t(errorCode(e) === 'DONATION_CERTIFIED' ? 'donations.certifiedLocked' : 'donations.actionFailed'));
        }
    };

    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>{t('donations.assign.title')}</DialogTitle>
            <DialogContent>
                <DialogContentText sx={{ mb: 2 }}>
                    {localDate(donation.date)} · {donation.description} · {money(donation.amount)}
                </DialogContentText>
                <Stack direction="row" gap={1} alignItems="flex-start">
                    <Autocomplete sx={{ flex: 1 }} options={options} value={party} onChange={(_, v) => setParty(v)}
                        groupBy={(o) => t(o.kind === 'member' ? 'donations.assign.members' : 'donations.assign.donors')}
                        getOptionLabel={partyLabel}
                        isOptionEqualToValue={(a, b) => partyLabel(a) === partyLabel(b) && a.kind === b.kind}
                        renderInput={(params) => <TextField {...params} label={t('donations.assign.donor')} />} />
                    <Tooltip title={t('donations.donors.create')}>
                        <IconButton onClick={onNewDonor} sx={{ mt: 1 }}><PersonAdd /></IconButton>
                    </Tooltip>
                </Stack>
                <FormControlLabel sx={{ mt: 1 }} control={<Checkbox checked={waiver} onChange={(e) => setWaiver(e.target.checked)} />}
                    label={t('donations.expenseWaiver')} />
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={save} disabled={!party}>{t('buttons.save')}</Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Donations ────────────────────────────────────────────────────────────────

function DonationsView({ editable }: { editable: boolean }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [donations, setDonations] = useState<Donation[] | null>(null);
    const [donors, setDonors] = useState<Donor[]>([]);
    const [members, setMembers] = useState<Member[]>([]);
    const [assigning, setAssigning] = useState<Donation | null>(null);
    const [editingDonor, setEditingDonor] = useState<Partial<Donor> | null>(null);

    const reload = useCallback(async () => {
        try {
            const [d, o] = await Promise.all([fetchDonations(), fetchDonors()]);
            setDonations(d);
            setDonors(o);
        } catch {
            addNotification(t('donations.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => {
        void reload();
        fetchMembers().then(setMembers).catch(() => setMembers([]));
    }, [reload]);

    if (donations === null) return <CircularProgress />;

    const unassigned = donations.filter((d) => !d.member && !d.donor).length;

    const unassign = async (d: Donation) => {
        try {
            await unassignDonation(d.transactionId);
        } catch (e) {
            addNotification(t(errorCode(e) === 'DONATION_CERTIFIED' ? 'donations.certifiedLocked' : 'donations.actionFailed'));
        }
        await reload();
    };

    const removeDonor = async (d: Donor) => {
        try {
            await deleteDonor(d.id);
        } catch (e) {
            addNotification(t(errorCode(e) === 'DONOR_IN_USE' ? 'donations.donors.inUse' : 'donations.actionFailed'));
        }
        await reload();
    };

    return (
        <Box>
            <Typography variant="body2" color="text.secondary" mb={2}>{t('donations.list.hint')}</Typography>
            {unassigned > 0 && <Alert severity="info" sx={{ mb: 2 }}>{t('donations.list.unassigned', { count: unassigned })}</Alert>}
            <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>{t('donations.columns.date')}</TableCell>
                            <TableCell>{t('donations.columns.description')}</TableCell>
                            <TableCell align="right">{t('donations.columns.amount')}</TableCell>
                            <TableCell>{t('donations.columns.donor')}</TableCell>
                            <TableCell>{t('donations.columns.receipt')}</TableCell>
                            {editable && <TableCell />}
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {donations.length === 0 && (
                            <TableRow><TableCell colSpan={6}><Typography variant="body2" color="text.secondary">
                                {t('donations.list.empty')}
                            </Typography></TableCell></TableRow>
                        )}
                        {donations.map((d) => (
                            <TableRow key={d.transactionId}>
                                <TableCell>{localDate(d.date)}</TableCell>
                                <TableCell>{d.description}</TableCell>
                                <TableCell align="right">{money(d.amount)}</TableCell>
                                <TableCell>
                                    {d.member ? `${d.member.firstName} ${d.member.lastName} (${d.member.number})`
                                        : d.donor ? d.donor.name
                                            : <Chip size="small" color="warning" label={t('donations.list.noDonor')} />}
                                    {d.expenseWaiver && <Chip size="small" variant="outlined" sx={{ ml: 1 }} label={t('donations.waiverShort')} />}
                                </TableCell>
                                <TableCell>{d.receiptNumber ?? '–'}</TableCell>
                                {editable && (
                                    <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                                        <Tooltip title={t('donations.assign.title')}>
                                            <span><IconButton size="small" color="primary" disabled={!!d.receiptNumber}
                                                onClick={() => setAssigning(d)}><Edit fontSize="small" /></IconButton></span>
                                        </Tooltip>
                                        {(d.member || d.donor) && (
                                            <Tooltip title={t('donations.unassign')}>
                                                <span><IconButton size="small" disabled={!!d.receiptNumber}
                                                    onClick={() => unassign(d)}><LinkOff fontSize="small" /></IconButton></span>
                                            </Tooltip>
                                        )}
                                    </TableCell>
                                )}
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>

            <Box display="flex" alignItems="center" justifyContent="space-between" mt={4} mb={1}>
                <Typography variant="h6">{t('donations.donors.title')}</Typography>
                {editable && (
                    <Button startIcon={<PersonAdd />} onClick={() => setEditingDonor({})}>{t('donations.donors.create')}</Button>
                )}
            </Box>
            <Typography variant="body2" color="text.secondary" mb={1}>{t('donations.donors.hint')}</Typography>
            <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                    <TableBody>
                        {donors.length === 0 && (
                            <TableRow><TableCell><Typography variant="body2" color="text.secondary">
                                {t('donations.donors.empty')}
                            </Typography></TableCell></TableRow>
                        )}
                        {donors.map((d) => (
                            <TableRow key={d.id}>
                                <TableCell>{d.name}</TableCell>
                                <TableCell>{[d.street, [d.postalCode, d.city].filter(Boolean).join(' ')].filter(Boolean).join(', ')}</TableCell>
                                {editable && (
                                    <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                                        <IconButton size="small" color="primary" onClick={() => setEditingDonor(d)}><Edit fontSize="small" /></IconButton>
                                        <IconButton size="small" onClick={() => removeDonor(d)}><Delete fontSize="small" /></IconButton>
                                    </TableCell>
                                )}
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>

            {assigning && (
                <AssignDialog donation={assigning} members={members} donors={donors}
                    onClose={() => setAssigning(null)}
                    onNewDonor={() => setEditingDonor({})}
                    onSaved={async () => { setAssigning(null); await reload(); }} />
            )}
            {editingDonor && (
                <DonorDialog donor={editingDonor} onClose={() => setEditingDonor(null)}
                    onSaved={async () => { setEditingDonor(null); await reload(); }} />
            )}
        </Box>
    );
}

// ─── Issuing receipts ─────────────────────────────────────────────────────────

function IssueView({ editable, settings, onIssued }: { editable: boolean; settings: DonationSettings | null; onIssued: () => void }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const currentYear = new Date().getFullYear();
    // Receipts for a year are usually issued in its first months.
    const [year, setYear] = useState(() => new Date().getMonth() < 3 ? currentYear - 1 : currentYear);
    const [issueDate, setIssueDate] = useState(() => isoDate(new Date()));
    const [candidates, setCandidates] = useState<DonationCandidate[] | null>(null);
    const [busy, setBusy] = useState(false);
    const [skipped, setSkipped] = useState<DonationCandidate[]>([]);

    const reload = useCallback(async () => {
        try {
            setCandidates(await fetchDonationCandidates(year));
        } catch {
            addNotification(t('donations.loadFailed'));
        }
    }, [year, addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const issue = async (fn: () => Promise<DonationReceipt[]>) => {
        setBusy(true);
        try {
            const created = await fn();
            if (created.length > 0) {
                const duplicates = created.filter((r) => r.duplicateOf).length;
                addNotification(duplicates > 0
                    ? t('donations.issue.duplicateCreated', { number: created[0].number })
                    : t('donations.issue.created', { count: created.length }));
                await downloadDonationReceipts(created);
            } else {
                addNotification(t('donations.issue.nothingOpen'));
            }
            onIssued();
        } catch (e) {
            const code = errorCode(e);
            addNotification(t(code === 'MISSING_ADDRESS' ? 'donations.issue.missingAddress'
                : code === 'RECEIPT_ITEMS_CERTIFIED' ? 'donations.issue.alreadyCertified'
                    : code === 'DONATIONS_NOT_CONFIGURED' ? 'donations.notConfigured' : 'donations.actionFailed'));
        } finally {
            setBusy(false);
        }
        await reload();
    };

    const issueFor = (c: DonationCandidate, refs: string[]) => issue(async () => [await createDonationReceipt({
        memberId: c.member?.id, donorId: c.donor?.id, items: refs, issueDate,
    })]);

    const annual = () => issue(async () => {
        const result = await createAnnualReceipts(year, issueDate);
        setSkipped(result.skipped);
        return result.created;
    });

    if (candidates === null) return <CircularProgress />;
    const open = candidates.filter((c) => c.items.some((it) => !it.receiptId));
    const disabled = !editable || !settings || busy;

    return (
        <Box>
            <Box display="flex" gap={2} flexWrap="wrap" alignItems="center" mb={2}>
                <TextField select size="small" label={t('donations.issue.year')} value={year} onChange={(e) => setYear(Number(e.target.value))}
                    sx={{ minWidth: 120 }}>
                    {Array.from({ length: 6 }, (_, i) => currentYear - i).map((y) => <MenuItem key={y} value={y}>{y}</MenuItem>)}
                </TextField>
                <TextField size="small" type="date" label={t('donations.issue.issueDate')} value={issueDate}
                    onChange={(e) => setIssueDate(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                <Box flex={1} />
                {editable && (
                    <Button variant="contained" startIcon={<ReceiptLong />} onClick={annual} disabled={disabled || open.length === 0}>
                        {t('donations.issue.annual', { count: open.length })}
                    </Button>
                )}
            </Box>
            <Typography variant="body2" color="text.secondary" mb={2}>
                {t(settings?.feesDeductible ? 'donations.issue.hintWithFees' : 'donations.issue.hint')}
            </Typography>
            {skipped.length > 0 && (
                <Alert severity="warning" sx={{ mb: 2 }} onClose={() => setSkipped([])}>
                    {t('donations.issue.skipped', { names: skipped.map((s) => s.name).join(', ') })}
                </Alert>
            )}
            {candidates.length === 0 && <Typography color="text.secondary">{t('donations.issue.empty', { year })}</Typography>}
            <Stack gap={2}>
                {candidates.map((c) => {
                    const openItems = c.items.filter((it) => !it.receiptId);
                    const total = c.items.reduce((s, it) => s + it.amount, 0);
                    return (
                        <Paper key={c.key} variant="outlined" sx={{ p: 2 }}>
                            <Box display="flex" justifyContent="space-between" gap={2} flexWrap="wrap" alignItems="flex-start">
                                <Box>
                                    <Typography fontWeight={600}>
                                        {c.name}{c.member ? ` (${c.member.number})` : ''}
                                        {c.donor && <Chip size="small" variant="outlined" sx={{ ml: 1 }} label={t('donations.external')} />}
                                    </Typography>
                                    {c.address
                                        ? <Typography variant="body2" color="text.secondary" sx={{ whiteSpace: 'pre-line' }}>{c.address}</Typography>
                                        : <Chip size="small" color="warning" label={t('donations.issue.noAddress')} sx={{ mt: 0.5 }} />}
                                </Box>
                                <Box textAlign="right">
                                    <Typography fontWeight={600}>{money(total)}</Typography>
                                    {editable && openItems.length > 0 && (
                                        <Button size="small" startIcon={<ReceiptLong />} disabled={disabled || !c.address}
                                            onClick={() => issueFor(c, openItems.map((it) => it.ref))}>
                                            {openItems.length === 1 ? t('donations.issue.single') : t('donations.issue.collective')}
                                        </Button>
                                    )}
                                </Box>
                            </Box>
                            <Table size="small" sx={{ mt: 1 }}>
                                <TableBody>
                                    {c.items.map((it) => (
                                        <TableRow key={it.ref}>
                                            <TableCell sx={{ width: 110 }}>{localDate(it.date)}</TableCell>
                                            <TableCell>
                                                {it.description}
                                                <Chip size="small" variant="outlined" sx={{ ml: 1 }} label={t(`donations.kind.${it.kind}`)} />
                                                {it.expenseWaiver && <Chip size="small" variant="outlined" sx={{ ml: 1 }} label={t('donations.waiverShort')} />}
                                            </TableCell>
                                            <TableCell align="right">{money(it.amount)}</TableCell>
                                            <TableCell sx={{ width: 180 }} align="right">
                                                {it.receiptNumber
                                                    ? <Chip size="small" color="success" label={t('donations.issue.certified', { number: it.receiptNumber })} />
                                                    : editable && openItems.length > 1 && (
                                                        <Button size="small" disabled={disabled || !c.address} onClick={() => issueFor(c, [it.ref])}>
                                                            {t('donations.issue.single')}
                                                        </Button>
                                                    )}
                                            </TableCell>
                                        </TableRow>
                                    ))}
                                </TableBody>
                            </Table>
                        </Paper>
                    );
                })}
            </Stack>
        </Box>
    );
}

// ─── Log ──────────────────────────────────────────────────────────────────────

function LogView({ editable, reloadKey }: { editable: boolean; reloadKey: number }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [receipts, setReceipts] = useState<DonationReceipt[] | null>(null);
    const [voiding, setVoiding] = useState<DonationReceipt | null>(null);
    const [reason, setReason] = useState('');

    const reload = useCallback(async () => {
        try {
            setReceipts(await fetchDonationReceipts());
        } catch {
            addNotification(t('donations.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload, reloadKey]);

    const act = async (fn: () => Promise<unknown>) => {
        try {
            await fn();
        } catch (e) {
            addNotification(t(errorCode(e) === 'RECEIPT_VOIDED' ? 'donations.log.voided' : 'donations.actionFailed'));
        }
        await reload();
    };

    if (receipts === null) return <CircularProgress />;
    const numbers = new Map(receipts.map((r) => [r.id, r.number]));

    return (
        <Box>
            <Typography variant="body2" color="text.secondary" mb={2}>{t('donations.log.hint')}</Typography>
            <TableContainer component={Paper} variant="outlined">
                <Table size="small">
                    <TableHead>
                        <TableRow>
                            <TableCell>{t('donations.log.number')}</TableCell>
                            <TableCell>{t('donations.log.issued')}</TableCell>
                            <TableCell>{t('donations.log.recipient')}</TableCell>
                            <TableCell>{t('donations.log.kind')}</TableCell>
                            <TableCell align="right">{t('donations.columns.amount')}</TableCell>
                            <TableCell>{t('donations.log.createdBy')}</TableCell>
                            <TableCell />
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {receipts.length === 0 && (
                            <TableRow><TableCell colSpan={7}><Typography variant="body2" color="text.secondary">
                                {t('donations.log.empty')}
                            </Typography></TableCell></TableRow>
                        )}
                        {receipts.map((r) => (
                            <TableRow key={r.id} sx={r.voidedAt ? { '& td': { color: 'text.disabled' } } : undefined}>
                                <TableCell sx={{ whiteSpace: 'nowrap' }}>
                                    {r.number}
                                    {r.duplicateOf && <Chip size="small" variant="outlined" sx={{ ml: 1 }}
                                        label={t('donations.log.duplicate', { number: numbers.get(r.duplicateOf) ?? r.number })} />}
                                    {r.voidedAt && (
                                        <Tooltip title={r.voidReason ?? ''}>
                                            <Chip size="small" color="error" variant="outlined" sx={{ ml: 1 }} label={t('donations.log.voidedChip')} />
                                        </Tooltip>
                                    )}
                                </TableCell>
                                <TableCell>{localDate(r.issueDate)}<Typography variant="caption" display="block" color="text.secondary">
                                    {new Date(r.createdAt).toLocaleString()}</Typography></TableCell>
                                <TableCell>{r.recipientName}</TableCell>
                                <TableCell>{t(`donations.receiptKind.${r.kind}`)} · {r.year}</TableCell>
                                <TableCell align="right">{money(r.total)}</TableCell>
                                <TableCell>{r.createdBy ?? '–'}</TableCell>
                                <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                                    <Tooltip title={t('donations.log.download')}>
                                        <IconButton size="small" color="primary"
                                            onClick={() => act(() => downloadDonationReceipts([r]))}><Download fontSize="small" /></IconButton>
                                    </Tooltip>
                                    {editable && !r.voidedAt && (
                                        <>
                                            <Tooltip title={t('donations.log.makeDuplicate')}>
                                                <IconButton size="small" onClick={() => act(async () => {
                                                    const dup = await duplicateDonationReceipt(r.id);
                                                    await downloadDonationReceipts([dup]);
                                                })}><ContentCopy fontSize="small" /></IconButton>
                                            </Tooltip>
                                            {!r.duplicateOf && (
                                                <Tooltip title={t('donations.log.void')}>
                                                    <IconButton size="small" onClick={() => { setReason(''); setVoiding(r); }}>
                                                        <Block fontSize="small" /></IconButton>
                                                </Tooltip>
                                            )}
                                        </>
                                    )}
                                </TableCell>
                            </TableRow>
                        ))}
                    </TableBody>
                </Table>
            </TableContainer>

            <Dialog open={!!voiding} onClose={() => setVoiding(null)} maxWidth="sm" fullWidth>
                <DialogTitle>{t('donations.log.voidTitle', { number: voiding?.number })}</DialogTitle>
                <DialogContent>
                    <DialogContentText mb={2}>{t('donations.log.voidText')}</DialogContentText>
                    <TextField fullWidth label={t('donations.log.voidReason')} value={reason} onChange={(e) => setReason(e.target.value)} />
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setVoiding(null)}>{t('buttons.abort')}</Button>
                    <Button color="error" variant="contained" disabled={!reason.trim()} onClick={async () => {
                        const r = voiding;
                        setVoiding(null);
                        if (r) await act(() => voidDonationReceipt(r.id, reason));
                    }}>{t('donations.log.void')}</Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
}

// ─── Tab ──────────────────────────────────────────────────────────────────────

type View = 'donations' | 'issue' | 'log';

/** The donations tab of the finance page. */
export function DonationReceiptsTab({ editable }: { editable: boolean }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [view, setView] = useState<View>('donations');
    const [settings, setSettings] = useState<DonationSettings | null | undefined>(undefined);
    const [issued, setIssued] = useState(0);

    useEffect(() => {
        fetchDonationSettings().then(setSettings).catch(() => {
            setSettings(null);
            addNotification(t('donations.loadFailed'));
        });
    }, [addNotification, t]);

    if (settings === undefined) return <CircularProgress />;

    return (
        <Box>
            {!settings && <Alert severity="info" sx={{ mb: 2 }}>{t('donations.notConfigured')}</Alert>}
            {settings?.exemptionExpired && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                    {t('donations.exemptionExpired', { date: localDate(settings.validUntil ?? settings.exemptionDate) })}
                </Alert>
            )}
            <ToggleButtonGroup size="small" exclusive value={view} onChange={(_, v) => v && setView(v)} sx={{ mb: 2 }}>
                <ToggleButton value="donations"><VolunteerActivism fontSize="small" sx={{ mr: 1 }} />{t('donations.views.donations')}</ToggleButton>
                <ToggleButton value="issue"><ReceiptLong fontSize="small" sx={{ mr: 1 }} />{t('donations.views.issue')}</ToggleButton>
                <ToggleButton value="log">{t('donations.views.log')}</ToggleButton>
            </ToggleButtonGroup>
            {view === 'donations' && <DonationsView editable={editable} />}
            {view === 'issue' && <IssueView editable={editable} settings={settings} onIssued={() => setIssued((n) => n + 1)} />}
            {view === 'log' && <LogView editable={editable} reloadKey={issued} />}
        </Box>
    );
}
