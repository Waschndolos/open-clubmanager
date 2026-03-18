import { useEffect, useState } from 'react';
import {
    Box,
    Button,
    Chip,
    CircularProgress,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputLabel,
    MenuItem,
    Select,
    Tab,
    Tabs,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { Add, AccountBalance, Delete, Edit, EuroSymbol } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import {
    fetchTransactions,
    createTransaction,
    updateTransaction,
    deleteTransaction,
    fetchMemberFees,
    createMemberFee,
    updateMemberFee,
    deleteMemberFee,
} from '../../api/finance';
import { fetchMembers } from '../../api/members';
import { FinanceTransaction, MemberFee, Member } from '../../api/types';
import PageHeader from '../../components/common/PageHeader';

// ─── Transaction Dialog ───────────────────────────────────────────────────────

type TransactionDialogProps = {
    transaction: Partial<FinanceTransaction>;
    isNew: boolean;
    onClose: () => void;
    onSave: (t: Partial<FinanceTransaction>) => void;
};

function TransactionDialog({ transaction, isNew, onClose, onSave }: TransactionDialogProps) {
    const { t } = useTranslation();
    const [form, setForm] = useState<Partial<FinanceTransaction>>(transaction);

    const handleChange = (field: keyof FinanceTransaction, value: string | number) => {
        setForm((prev) => ({ ...prev, [field]: value }));
    };

    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {isNew ? t('finance.transactions.dialog.titleCreate') : t('finance.transactions.dialog.titleEdit')}
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField
                        label={t('finance.transactions.fields.date')}
                        type="date"
                        value={form.date ? form.date.slice(0, 10) : ''}
                        onChange={(e) => handleChange('date', e.target.value)}
                        slotProps={{ inputLabel: { shrink: true } }}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.transactions.fields.description')}
                        value={form.description ?? ''}
                        onChange={(e) => handleChange('description', e.target.value)}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.transactions.fields.amount')}
                        type="number"
                        value={form.amount ?? ''}
                        onChange={(e) => handleChange('amount', e.target.value === '' ? 0 : parseFloat(e.target.value))}
                        fullWidth
                    />
                    <FormControl fullWidth>
                        <InputLabel>{t('finance.transactions.fields.type')}</InputLabel>
                        <Select
                            label={t('finance.transactions.fields.type')}
                            value={form.type ?? ''}
                            onChange={(e) => handleChange('type', e.target.value)}
                        >
                            <MenuItem value="income">{t('finance.transactions.types.income')}</MenuItem>
                            <MenuItem value="expense">{t('finance.transactions.types.expense')}</MenuItem>
                        </Select>
                    </FormControl>
                    <TextField
                        label={t('finance.transactions.fields.category')}
                        value={form.category ?? ''}
                        onChange={(e) => handleChange('category', e.target.value)}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.transactions.fields.notes')}
                        value={form.notes ?? ''}
                        onChange={(e) => handleChange('notes', e.target.value)}
                        multiline
                        rows={2}
                        fullWidth
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button
                    variant="contained"
                    onClick={() => onSave(form)}
                    disabled={!form.date || !form.description || form.amount === undefined || !form.type}
                >
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Member Fee Dialog ────────────────────────────────────────────────────────

type MemberFeeDialogProps = {
    fee: Partial<MemberFee>;
    members: Member[];
    isNew: boolean;
    onClose: () => void;
    onSave: (f: Partial<MemberFee>) => void;
};

function MemberFeeDialog({ fee, members, isNew, onClose, onSave }: MemberFeeDialogProps) {
    const { t } = useTranslation();
    const [form, setForm] = useState<Partial<MemberFee>>(fee);

    const handleChange = (field: keyof MemberFee, value: string | number | null) => {
        setForm((prev) => ({ ...prev, [field]: value }));
    };

    return (
        <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle>
                {isNew ? t('finance.memberfees.dialog.titleCreate') : t('finance.memberfees.dialog.titleEdit')}
            </DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <FormControl fullWidth>
                        <InputLabel>{t('finance.memberfees.fields.member')}</InputLabel>
                        <Select
                            label={t('finance.memberfees.fields.member')}
                            value={form.memberId ?? ''}
                            onChange={(e) => handleChange('memberId', Number(e.target.value))}
                        >
                            {members.map((m) => (
                                <MenuItem key={m.id} value={m.id}>
                                    {m.firstName} {m.lastName} ({m.number})
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <TextField
                        label={t('finance.memberfees.fields.amount')}
                        type="number"
                        value={form.amount ?? ''}
                        onChange={(e) => handleChange('amount', e.target.value === '' ? 0 : parseFloat(e.target.value))}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.memberfees.fields.year')}
                        type="number"
                        value={form.year ?? new Date().getFullYear()}
                        onChange={(e) => handleChange('year', e.target.value === '' ? new Date().getFullYear() : parseInt(e.target.value))}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.memberfees.fields.dueDate')}
                        type="date"
                        value={form.dueDate ? form.dueDate.slice(0, 10) : ''}
                        onChange={(e) => handleChange('dueDate', e.target.value)}
                        slotProps={{ inputLabel: { shrink: true } }}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.memberfees.fields.paidDate')}
                        type="date"
                        value={form.paidDate ? form.paidDate.slice(0, 10) : ''}
                        onChange={(e) => handleChange('paidDate', e.target.value || null)}
                        slotProps={{ inputLabel: { shrink: true } }}
                        fullWidth
                    />
                    <TextField
                        label={t('finance.memberfees.fields.description')}
                        value={form.description ?? ''}
                        onChange={(e) => handleChange('description', e.target.value)}
                        fullWidth
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button
                    variant="contained"
                    onClick={() => onSave(form)}
                    disabled={!form.memberId || form.amount === undefined || !form.dueDate || !form.year}
                >
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

// ─── Balance Summary ──────────────────────────────────────────────────────────

function BalanceSummary({ transactions, fees }: { transactions: FinanceTransaction[]; fees: MemberFee[] }) {
    const { t } = useTranslation();

    const totalIncome = transactions
        .filter((tx) => tx.type === 'income')
        .reduce((sum, tx) => sum + tx.amount, 0);
    const totalExpense = transactions
        .filter((tx) => tx.type === 'expense')
        .reduce((sum, tx) => sum + tx.amount, 0);
    const balance = totalIncome - totalExpense;

    const totalFeesDue = fees.reduce((sum, f) => sum + f.amount, 0);
    const totalFeesPaid = fees.filter((f) => f.paidDate).reduce((sum, f) => sum + f.amount, 0);
    const totalFeesOutstanding = totalFeesDue - totalFeesPaid;

    const fmt = (n: number) =>
        n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });

    return (
        <Box display="flex" gap={2} flexWrap="wrap" mb={3}>
            {[
                { label: t('finance.summary.balance'), value: fmt(balance), color: balance >= 0 ? 'success.main' : 'error.main' },
                { label: t('finance.summary.totalIncome'), value: fmt(totalIncome), color: 'success.main' },
                { label: t('finance.summary.totalExpense'), value: fmt(totalExpense), color: 'error.main' },
                { label: t('finance.summary.feesOutstanding'), value: fmt(totalFeesOutstanding), color: totalFeesOutstanding > 0 ? 'warning.main' : 'success.main' },
            ].map(({ label, value, color }) => (
                <Box
                    key={label}
                    sx={{
                        flex: '1 1 150px',
                        p: 2,
                        borderRadius: 2,
                        border: '1px solid',
                        borderColor: 'divider',
                        bgcolor: 'background.paper',
                    }}
                >
                    <Typography variant="body2" color="text.secondary">{label}</Typography>
                    <Typography variant="h6" fontWeight={700} color={color}>{value} €</Typography>
                </Box>
            ))}
        </Box>
    );
}

// ─── Finance Page ─────────────────────────────────────────────────────────────

export function Finance() {
    const { t } = useTranslation();
    const [tab, setTab] = useState(0);
    const [loading, setLoading] = useState(true);
    const [transactions, setTransactions] = useState<FinanceTransaction[]>([]);
    const [fees, setFees] = useState<MemberFee[]>([]);
    const [members, setMembers] = useState<Member[]>([]);

    const [txDialog, setTxDialog] = useState<{ open: boolean; item: Partial<FinanceTransaction>; isNew: boolean }>({
        open: false,
        item: {},
        isNew: true,
    });
    const [feeDialog, setFeeDialog] = useState<{ open: boolean; item: Partial<MemberFee>; isNew: boolean }>({
        open: false,
        item: {},
        isNew: true,
    });

    useEffect(() => {
        Promise.all([fetchTransactions(), fetchMemberFees(), fetchMembers()])
            .then(([txs, mf, ms]) => {
                setTransactions(txs);
                setFees(mf);
                setMembers(ms);
            })
            .finally(() => setLoading(false));
    }, []);

    // ── Transaction handlers ──────────────────────────────────────────────────

    const handleSaveTransaction = async (data: Partial<FinanceTransaction>) => {
        if (txDialog.isNew) {
            const created = await createTransaction(data as Omit<FinanceTransaction, 'id' | 'createdAt' | 'updatedAt'>);
            setTransactions((prev) => [created, ...prev]);
        } else {
            const updated = await updateTransaction(data as FinanceTransaction);
            setTransactions((prev) => prev.map((tx) => (tx.id === updated.id ? updated : tx)));
        }
        setTxDialog({ open: false, item: {}, isNew: true });
    };

    const handleDeleteTransaction = async (id: number) => {
        await deleteTransaction(id);
        setTransactions((prev) => prev.filter((tx) => tx.id !== id));
    };

    // ── Member fee handlers ───────────────────────────────────────────────────

    const handleSaveFee = async (data: Partial<MemberFee>) => {
        if (feeDialog.isNew) {
            const created = await createMemberFee(data as Omit<MemberFee, 'id' | 'member' | 'createdAt' | 'updatedAt'>);
            setFees((prev) => [created, ...prev]);
        } else {
            const updated = await updateMemberFee(data as MemberFee);
            setFees((prev) => prev.map((f) => (f.id === updated.id ? updated : f)));
        }
        setFeeDialog({ open: false, item: {}, isNew: true });
    };

    const handleDeleteFee = async (id: number) => {
        await deleteMemberFee(id);
        setFees((prev) => prev.filter((f) => f.id !== id));
    };

    // ── Column definitions ────────────────────────────────────────────────────

    const transactionColumns: GridColDef[] = [
        {
            field: 'date',
            headerName: t('finance.transactions.fields.date'),
            flex: 1,
            minWidth: 110,
            valueFormatter: (value: string) => value ? new Date(value).toLocaleDateString() : '',
        },
        { field: 'description', headerName: t('finance.transactions.fields.description'), flex: 2, minWidth: 180 },
        {
            field: 'amount',
            headerName: t('finance.transactions.fields.amount'),
            flex: 1,
            minWidth: 100,
            valueFormatter: (value: number) =>
                value !== undefined ? `${value.toFixed(2)} €` : '',
        },
        {
            field: 'type',
            headerName: t('finance.transactions.fields.type'),
            flex: 1,
            minWidth: 100,
            renderCell: (params) => (
                <Chip
                    size="small"
                    label={params.value === 'income' ? t('finance.transactions.types.income') : t('finance.transactions.types.expense')}
                    color={params.value === 'income' ? 'success' : 'error'}
                />
            ),
        },
        { field: 'category', headerName: t('finance.transactions.fields.category'), flex: 1, minWidth: 120 },
        {
            field: 'actions',
            headerName: t('members.table.header.actions'),
            width: 100,
            sortable: false,
            renderCell: (params) => (
                <Box>
                    <Tooltip title={t('tooltips.edit')}>
                        <IconButton
                            size="small"
                            color="primary"
                            onClick={() => setTxDialog({ open: true, item: params.row as FinanceTransaction, isNew: false })}
                        >
                            <Edit fontSize="small" />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t('tooltips.delete')}>
                        <IconButton
                            size="small"
                            color="primary"
                            onClick={() => handleDeleteTransaction((params.row as FinanceTransaction).id)}
                        >
                            <Delete fontSize="small" />
                        </IconButton>
                    </Tooltip>
                </Box>
            ),
        },
    ];

    const feeColumns: GridColDef[] = [
        {
            field: 'member',
            headerName: t('finance.memberfees.fields.member'),
            flex: 2,
            minWidth: 160,
            valueGetter: (_value, row) => {
                const fee = row as MemberFee;
                return fee.member ? `${fee.member.firstName} ${fee.member.lastName} (${fee.member.number})` : fee.memberId;
            },
        },
        { field: 'year', headerName: t('finance.memberfees.fields.year'), flex: 0.5, minWidth: 80 },
        {
            field: 'amount',
            headerName: t('finance.memberfees.fields.amount'),
            flex: 1,
            minWidth: 100,
            valueFormatter: (value: number) =>
                value !== undefined ? `${value.toFixed(2)} €` : '',
        },
        {
            field: 'dueDate',
            headerName: t('finance.memberfees.fields.dueDate'),
            flex: 1,
            minWidth: 110,
            valueFormatter: (value: string) => value ? new Date(value).toLocaleDateString() : '',
        },
        {
            field: 'paidDate',
            headerName: t('finance.memberfees.fields.paidDate'),
            flex: 1,
            minWidth: 110,
            renderCell: (params) =>
                params.value ? (
                    <Chip size="small" label={new Date(params.value as string).toLocaleDateString()} color="success" />
                ) : (
                    <Chip size="small" label={t('finance.memberfees.unpaid')} color="warning" />
                ),
        },
        { field: 'description', headerName: t('finance.memberfees.fields.description'), flex: 1.5, minWidth: 140 },
        {
            field: 'actions',
            headerName: t('members.table.header.actions'),
            width: 100,
            sortable: false,
            renderCell: (params) => (
                <Box>
                    <Tooltip title={t('tooltips.edit')}>
                        <IconButton
                            size="small"
                            color="primary"
                            onClick={() => setFeeDialog({ open: true, item: params.row as MemberFee, isNew: false })}
                        >
                            <Edit fontSize="small" />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t('tooltips.delete')}>
                        <IconButton
                            size="small"
                            color="primary"
                            onClick={() => handleDeleteFee((params.row as MemberFee).id)}
                        >
                            <Delete fontSize="small" />
                        </IconButton>
                    </Tooltip>
                </Box>
            ),
        },
    ];

    if (loading) return <CircularProgress />;

    return (
        <Box p={3}>
            <PageHeader
                title={t('finance.title')}
                icon={<AccountBalance fontSize="small" />}
            />

            <BalanceSummary transactions={transactions} fees={fees} />

            <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 2 }}>
                <Tab label={t('finance.tabs.transactions')} />
                <Tab label={t('finance.tabs.memberfees')} />
            </Tabs>

            {tab === 0 && (
                <Box>
                    <Box display="flex" justifyContent="flex-end" mb={1}>
                        <Button
                            variant="contained"
                            startIcon={<Add />}
                            onClick={() =>
                                setTxDialog({
                                    open: true,
                                    item: { date: new Date().toISOString(), type: 'income' },
                                    isNew: true,
                                })
                            }
                        >
                            {t('finance.transactions.create')}
                        </Button>
                    </Box>
                    <DataGrid
                        autoHeight
                        rows={transactions}
                        columns={transactionColumns}
                        pageSizeOptions={[10, 25, 50]}
                    />
                </Box>
            )}

            {tab === 1 && (
                <Box>
                    <Box display="flex" justifyContent="flex-end" mb={1}>
                        <Button
                            variant="contained"
                            startIcon={<EuroSymbol />}
                            onClick={() =>
                                setFeeDialog({
                                    open: true,
                                    item: {
                                        year: new Date().getFullYear(),
                                        dueDate: new Date().toISOString(),
                                    },
                                    isNew: true,
                                })
                            }
                        >
                            {t('finance.memberfees.create')}
                        </Button>
                    </Box>
                    <DataGrid
                        autoHeight
                        rows={fees}
                        columns={feeColumns}
                        pageSizeOptions={[10, 25, 50]}
                    />
                </Box>
            )}

            {txDialog.open && (
                <TransactionDialog
                    transaction={txDialog.item}
                    isNew={txDialog.isNew}
                    onClose={() => setTxDialog({ open: false, item: {}, isNew: true })}
                    onSave={handleSaveTransaction}
                />
            )}

            {feeDialog.open && (
                <MemberFeeDialog
                    fee={feeDialog.item}
                    members={members}
                    isNew={feeDialog.isNew}
                    onClose={() => setFeeDialog({ open: false, item: {}, isNew: true })}
                    onSave={handleSaveFee}
                />
            )}
        </Box>
    );
}
