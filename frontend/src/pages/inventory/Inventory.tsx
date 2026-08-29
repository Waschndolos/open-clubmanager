import { useEffect, useMemo, useState } from 'react';
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputAdornment,
    InputLabel,
    MenuItem,
    Select,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { Add, Clear, Edit, Delete, Inventory2 } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { Member, InventoryItem, InventoryLoan } from '../../api/types';
import { fetchMembers } from '../../api/members';
import {
    createInventoryItem,
    createInventoryLoan,
    deleteInventoryItem,
    fetchInventoryItems,
    fetchInventoryLoans,
    updateInventoryItem,
    updateInventoryLoan,
} from '../../api/inventory';

type ItemDialogProps = {
    item: Partial<InventoryItem>;
    isNew: boolean;
    onClose: () => void;
    onSave: (item: Partial<InventoryItem>) => Promise<void>;
};

function ItemDialog({ item, isNew, onClose, onSave }: ItemDialogProps) {
    const { t } = useTranslation();
    const [form, setForm] = useState<Partial<InventoryItem>>(item);

    const setField = (field: keyof InventoryItem, value: string | number | null) => {
        setForm((prev) => ({ ...prev, [field]: value }));
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{isNew ? t('inventory.items.create') : t('inventory.items.edit')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField label={t('inventory.fields.name')} value={form.name ?? ''} onChange={(e) => setField('name', e.target.value)} fullWidth />
                    <TextField label={t('inventory.fields.description')} value={form.description ?? ''} onChange={(e) => setField('description', e.target.value)} fullWidth />
                    <TextField label={t('inventory.fields.serialNumber')} value={form.serialNumber ?? ''} onChange={(e) => setField('serialNumber', e.target.value)} fullWidth />
                    <TextField label={t('inventory.fields.category')} value={form.category ?? ''} onChange={(e) => setField('category', e.target.value)} fullWidth />
                    <TextField label={t('inventory.fields.quantity')} type="number" value={form.quantity ?? 1} onChange={(e) => setField('quantity', Number(e.target.value))} fullWidth />
                    <TextField label={t('inventory.fields.location')} value={form.location ?? ''} onChange={(e) => setField('location', e.target.value)} fullWidth />
                    <TextField
                        label={t('inventory.fields.purchaseDate')}
                        type="date"
                        value={form.purchaseDate ? form.purchaseDate.slice(0, 10) : ''}
                        onChange={(e) => setField('purchaseDate', e.target.value || null)}
                        slotProps={{ inputLabel: { shrink: true } }}
                        fullWidth
                    />
                    <TextField
                        label={t('inventory.fields.purchasePrice')}
                        type="number"
                        value={form.purchasePrice ?? ''}
                        onChange={(e) => setField('purchasePrice', e.target.value === '' ? null : Number(e.target.value))}
                        fullWidth
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button
                    variant="contained"
                    onClick={() => void onSave(form)}
                    disabled={!form.name || !form.category || form.quantity === undefined || !form.location}
                >
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

type LoanDialogProps = {
    items: InventoryItem[];
    members: Member[];
    onClose: () => void;
    onSave: (loan: Partial<InventoryLoan>) => Promise<void>;
};

function LoanDialog({ items, members, onClose, onSave }: LoanDialogProps) {
    const { t } = useTranslation();
    const [form, setForm] = useState<Partial<InventoryLoan>>({
        loanedAt: new Date().toISOString(),
        notes: '',
    });

    const setField = (field: keyof InventoryLoan, value: string | number | null) => {
        setForm((prev) => ({ ...prev, [field]: value }));
    };

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{t('inventory.loans.create')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <FormControl fullWidth>
                        <InputLabel>{t('inventory.fields.item')}</InputLabel>
                        <Select
                            label={t('inventory.fields.item')}
                            value={form.itemId ?? ''}
                            onChange={(e) => setField('itemId', Number(e.target.value))}
                        >
                            {items.map((item) => (
                                <MenuItem key={item.id} value={item.id}>{item.name}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <FormControl fullWidth>
                        <InputLabel>{t('inventory.fields.member')}</InputLabel>
                        <Select
                            label={t('inventory.fields.member')}
                            value={form.memberId ?? ''}
                            onChange={(e) => setField('memberId', Number(e.target.value))}
                        >
                            {members.map((member) => (
                                <MenuItem key={member.id} value={member.id}>
                                    {member.firstName} {member.lastName} ({member.number})
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <TextField
                        label={t('inventory.fields.dueDate')}
                        type="date"
                        value={form.dueDate ? form.dueDate.slice(0, 10) : ''}
                        onChange={(e) => setField('dueDate', e.target.value || null)}
                        slotProps={{ inputLabel: { shrink: true } }}
                        fullWidth
                    />
                    <TextField
                        label={t('inventory.fields.notes')}
                        value={form.notes ?? ''}
                        onChange={(e) => setField('notes', e.target.value)}
                        multiline
                        rows={2}
                        fullWidth
                    />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" onClick={() => void onSave(form)} disabled={!form.itemId || !form.memberId}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

export default function Inventory() {
    const { t } = useTranslation();
    const [items, setItems] = useState<InventoryItem[]>([]);
    const [loans, setLoans] = useState<InventoryLoan[]>([]);
    const [members, setMembers] = useState<Member[]>([]);
    const [search, setSearch] = useState('');
    const [categoryFilter, setCategoryFilter] = useState('ALL');
    const [itemDialog, setItemDialog] = useState<{ open: boolean; isNew: boolean; item: Partial<InventoryItem> }>({
        open: false,
        isNew: true,
        item: {},
    });
    const [loanDialogOpen, setLoanDialogOpen] = useState(false);

    useEffect(() => {
        Promise.all([fetchInventoryItems(), fetchInventoryLoans(), fetchMembers()]).then(([itemRows, loanRows, memberRows]) => {
            setItems(itemRows);
            setLoans(loanRows);
            setMembers(memberRows);
        });
    }, []);

    const categories = useMemo(
        () => Array.from(new Set(items.map((item) => item.category))).sort((a, b) => a.localeCompare(b)),
        [items]
    );

    const filteredItems = useMemo(() => {
        const query = search.trim().toLowerCase();
        return items.filter((item) => {
            const matchesCategory = categoryFilter === 'ALL' || item.category === categoryFilter;
            const matchesSearch =
                query === '' ||
                item.name.toLowerCase().includes(query) ||
                item.category.toLowerCase().includes(query) ||
                item.location.toLowerCase().includes(query) ||
                (item.serialNumber ?? '').toLowerCase().includes(query);
            return matchesCategory && matchesSearch;
        });
    }, [items, search, categoryFilter]);

    const activeLoans = useMemo(() => loans.filter((loan) => !loan.returnedAt), [loans]);
    const itemById = useMemo(() => new Map(items.map((item) => [item.id, item])), [items]);
    const memberById = useMemo(() => new Map(members.map((member) => [member.id, member])), [members]);

    const saveItem = async (item: Partial<InventoryItem>) => {
        if (itemDialog.isNew) {
            const created = await createInventoryItem(item as Omit<InventoryItem, 'id' | 'createdAt' | 'updatedAt'>);
            setItems((prev) => [...prev, created]);
        } else if (itemDialog.item.id) {
            const updated = await updateInventoryItem(itemDialog.item.id, item);
            setItems((prev) => prev.map((entry) => (entry.id === updated.id ? updated : entry)));
        }
        setItemDialog({ open: false, isNew: true, item: {} });
    };

    const saveLoan = async (loan: Partial<InventoryLoan>) => {
        const created = await createInventoryLoan({
            itemId: loan.itemId as number,
            memberId: loan.memberId as number,
            loanedAt: (loan.loanedAt as string) || new Date().toISOString(),
            dueDate: loan.dueDate ?? null,
            returnedAt: null,
            notes: (loan.notes as string) || '',
        });
        setLoans((prev) => [created, ...prev]);
        setLoanDialogOpen(false);
    };

    const markReturned = async (loan: InventoryLoan) => {
        const updated = await updateInventoryLoan(loan.id, { returnedAt: new Date().toISOString() });
        setLoans((prev) => prev.map((entry) => (entry.id === updated.id ? updated : entry)));
    };

    const itemColumns: GridColDef[] = [
        { field: 'name', headerName: t('inventory.fields.name'), flex: 1.5, minWidth: 160 },
        { field: 'category', headerName: t('inventory.fields.category'), flex: 1, minWidth: 120 },
        { field: 'quantity', headerName: t('inventory.fields.quantity'), flex: 0.6, minWidth: 90 },
        { field: 'location', headerName: t('inventory.fields.location'), flex: 1, minWidth: 120 },
        { field: 'serialNumber', headerName: t('inventory.fields.serialNumber'), flex: 1, minWidth: 140 },
        {
            field: 'actions',
            headerName: t('members.table.header.actions'),
            sortable: false,
            width: 100,
            renderCell: (params) => (
                <Box>
                    <Tooltip title={t('tooltips.edit')}>
                        <IconButton size="small" onClick={() => setItemDialog({ open: true, isNew: false, item: params.row as InventoryItem })}>
                            <Edit fontSize="small" />
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t('tooltips.delete')}>
                        <IconButton
                            size="small"
                            onClick={async () => {
                                const id = (params.row as InventoryItem).id;
                                await deleteInventoryItem(id);
                                setItems((prev) => prev.filter((entry) => entry.id !== id));
                            }}
                        >
                            <Delete fontSize="small" />
                        </IconButton>
                    </Tooltip>
                </Box>
            ),
        },
    ];

    const loanColumns: GridColDef[] = [
        {
            field: 'itemId',
            headerName: t('inventory.fields.item'),
            flex: 1.2,
            minWidth: 160,
            valueGetter: (value: number) => itemById.get(value)?.name ?? value,
        },
        {
            field: 'memberId',
            headerName: t('inventory.fields.member'),
            flex: 1.4,
            minWidth: 190,
            valueGetter: (value: number) => {
                const member = memberById.get(value);
                return member ? `${member.firstName} ${member.lastName}` : value;
            },
        },
        {
            field: 'loanedAt',
            headerName: t('inventory.fields.loanedAt'),
            flex: 1,
            minWidth: 120,
            valueFormatter: (value: string) => value ? new Date(value).toLocaleDateString() : '',
        },
        {
            field: 'dueDate',
            headerName: t('inventory.fields.dueDate'),
            flex: 1,
            minWidth: 120,
            valueFormatter: (value: string | null) => value ? new Date(value).toLocaleDateString() : '—',
        },
        { field: 'notes', headerName: t('inventory.fields.notes'), flex: 1.2, minWidth: 140 },
        {
            field: 'actions',
            headerName: t('members.table.header.actions'),
            sortable: false,
            width: 150,
            renderCell: (params) => (
                <Button size="small" variant="outlined" onClick={() => void markReturned(params.row as InventoryLoan)}>
                    {t('inventory.loans.markReturned')}
                </Button>
            ),
        },
    ];

    return (
        <Box p={3}>
            <PageHeader
                title={t('inventory.title')}
                icon={<Inventory2 fontSize="small" />}
                actions={
                    <>
                        <Button variant="outlined" onClick={() => setLoanDialogOpen(true)}>
                            {t('inventory.loans.create')}
                        </Button>
                        <Button
                            variant="contained"
                            startIcon={<Add />}
                            onClick={() => setItemDialog({ open: true, isNew: true, item: { quantity: 1 } })}
                        >
                            {t('inventory.items.create')}
                        </Button>
                    </>
                }
            />

            <Box display="flex" gap={2} mb={2} flexWrap="wrap">
                <TextField
                    label={t('inventory.search')}
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    size="small"
                    slotProps={{
                        input: {
                            endAdornment: search && (
                                <InputAdornment position="end">
                                    <IconButton onClick={() => setSearch('')} size="small">
                                        <Clear fontSize="small" />
                                    </IconButton>
                                </InputAdornment>
                            ),
                        },
                    }}
                />
                <FormControl size="small" sx={{ minWidth: 220 }}>
                    <InputLabel>{t('inventory.categoryFilter')}</InputLabel>
                    <Select
                        label={t('inventory.categoryFilter')}
                        value={categoryFilter}
                        onChange={(e) => setCategoryFilter(e.target.value)}
                    >
                        <MenuItem value="ALL">{t('inventory.categoryAll')}</MenuItem>
                        {categories.map((category) => (
                            <MenuItem key={category} value={category}>{category}</MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </Box>

            <DataGrid autoHeight rows={filteredItems} columns={itemColumns} pageSizeOptions={[10, 25, 50]} />

            <Typography variant="h6" mt={4} mb={1}>{t('inventory.loans.activeTitle')}</Typography>
            <DataGrid autoHeight rows={activeLoans} columns={loanColumns} pageSizeOptions={[10, 25, 50]} />

            {itemDialog.open && (
                <ItemDialog
                    item={itemDialog.item}
                    isNew={itemDialog.isNew}
                    onClose={() => setItemDialog({ open: false, isNew: true, item: {} })}
                    onSave={saveItem}
                />
            )}
            {loanDialogOpen && (
                <LoanDialog
                    items={items}
                    members={members}
                    onClose={() => setLoanDialogOpen(false)}
                    onSave={saveLoan}
                />
            )}
        </Box>
    );
}
