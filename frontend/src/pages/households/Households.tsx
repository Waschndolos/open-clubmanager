import { useCallback, useEffect, useState } from 'react';
import {
    Alert,
    Autocomplete,
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    IconButton,
    MenuItem,
    Switch,
    TextField,
    Tooltip,
} from '@mui/material';
import { Add, Delete, Edit, FamilyRestroomOutlined, Star, WarningAmber } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { isAxiosError } from 'axios';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import PageHeader from '../../components/common/PageHeader';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';
import { fetchMembers } from '../../api/members';
import { Member } from '../../api/types';
import {
    createHousehold,
    deleteHousehold,
    fetchHouseholds,
    Household,
    HouseholdInput,
    memberName,
    updateHousehold,
} from '../../api/households';

export default function Households() {
    const { t } = useTranslation();
    const { appRole } = useAuth();
    const { addNotification } = useNotification();
    const navigate = useNavigate();
    const area = usePermission('membersCore');
    const canEdit = appRole !== 'READONLY' && area === 'write';

    const [households, setHouseholds] = useState<Household[]>([]);
    const [editing, setEditing] = useState<Partial<Household> | null>(null);

    const reload = useCallback(async () => {
        try {
            setHouseholds(await fetchHouseholds());
        } catch {
            addNotification(t('households.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const run = async (fn: () => Promise<unknown>) => {
        try {
            await fn();
        } catch (e) {
            const inOther = isAxiosError(e) && e.response?.data?.code === 'MEMBER_IN_OTHER_HOUSEHOLD';
            addNotification(t(inOther ? 'households.memberInOther' : 'households.saveFailed'));
        }
        await reload();
    };

    const cols: GridColDef<Household>[] = [
        {
            field: 'name', headerName: t('households.columns.name'), flex: 1, minWidth: 180,
            renderCell: ({ row }) => (
                <Box display="flex" alignItems="center" gap={1}>
                    {row.name}
                    {row.payerMissing && (
                        <Tooltip title={t('households.payerMissing')}>
                            <WarningAmber color="warning" fontSize="small" aria-label={t('households.payerMissing')} />
                        </Tooltip>
                    )}
                </Box>
            ),
        },
        {
            field: 'members', headerName: t('households.columns.members'), flex: 2, minWidth: 280, sortable: false,
            renderCell: ({ row }) => (
                <Box display="flex" flexWrap="wrap" gap={0.5} alignItems="center" height="100%">
                    {row.members.map((m) => (
                        <Chip key={m.id} size="small" clickable label={memberName(m)}
                              variant={m.active ? 'filled' : 'outlined'}
                              icon={row.payer?.id === m.id ? <Star fontSize="small" /> : undefined}
                              title={m.active ? undefined : t('households.left')}
                              sx={m.active ? undefined : { textDecoration: 'line-through' }}
                              onClick={() => navigate(`/members?edit=${encodeURIComponent(m.id)}`)} />
                    ))}
                </Box>
            ),
        },
        {
            field: 'payer', headerName: t('households.columns.payer'), flex: 1, minWidth: 160,
            valueGetter: (_v, r) => r.payer ? memberName(r.payer) : '–',
        },
        {
            field: 'sharing', headerName: t('households.columns.sharing'), width: 200, sortable: false,
            valueGetter: (_v, r) => [r.shareAddress && t('households.shareAddress'), r.shareBank && t('households.shareBank')]
                .filter(Boolean).join(', '),
        },
        ...(canEdit ? [{
            field: 'actions', headerName: '', width: 110, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <>
                    <IconButton aria-label={t('households.edit')} onClick={() => setEditing(row)}><Edit /></IconButton>
                    <IconButton aria-label={t('households.delete')} onClick={() => {
                        if (window.confirm(t('households.confirmDelete'))) void run(() => deleteHousehold(row.id));
                    }}><Delete /></IconButton>
                </>
            ),
        } as GridColDef<Household>] : []),
    ];

    const missing = households.filter((h) => h.payerMissing);

    return (
        <Box p={3}>
            <PageHeader
                title={t('households.title')}
                subtitle={t('households.subtitle')}
                icon={<FamilyRestroomOutlined fontSize="small" />}
                actions={canEdit && (
                    <Button variant="contained" startIcon={<Add />}
                            onClick={() => setEditing({ shareAddress: true, shareBank: false, members: [] })}>
                        {t('households.add')}
                    </Button>
                )}
            />
            {missing.length > 0 && (
                <Alert severity="warning" sx={{ mb: 2 }}>
                    {t('households.payerMissingList', { names: missing.map((h) => h.name).join(', ') })}
                </Alert>
            )}
            <DataGrid rows={households} columns={cols} autoHeight disableRowSelectionOnClick getRowHeight={() => 'auto'}
                      sx={{ '& .MuiDataGrid-cell': { py: 1, display: 'flex', alignItems: 'center' } }}
                      localeText={{ noRowsLabel: t('households.empty') }} />
            {editing && (
                <HouseholdDialog household={editing} onClose={() => setEditing(null)} onSave={async (input) => {
                    const id = editing.id;
                    setEditing(null);
                    await run(() => (id ? updateHousehold(id, input) : createHousehold(input)));
                }} />
            )}
        </Box>
    );
}

function HouseholdDialog({ household, onClose, onSave }: {
    household: Partial<Household>;
    onClose: () => void;
    onSave: (h: HouseholdInput) => Promise<void>;
}) {
    const { t } = useTranslation();
    const [members, setMembers] = useState<Member[]>([]);
    const [name, setName] = useState(household.name ?? '');
    const [memberIds, setMemberIds] = useState<string[]>(household.members?.map((m) => m.id) ?? []);
    const [payerId, setPayerId] = useState(household.payer?.id ?? '');
    const [shareAddress, setShareAddress] = useState(household.shareAddress ?? false);
    const [shareBank, setShareBank] = useState(household.shareBank ?? false);

    useEffect(() => {
        fetchMembers().then(setMembers).catch(() => undefined);
    }, []);

    const label = (m: Member) => `${m.firstName} ${m.lastName} (${m.number})`;
    const selected = memberIds.map((id) => members.find((m) => m.id === id)).filter((m): m is Member => !!m);

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{household.id ? t('households.edit') : t('households.add')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField autoFocus required label={t('households.columns.name')} value={name}
                               placeholder={t('households.namePlaceholder')} onChange={(e) => setName(e.target.value)} />
                    <Autocomplete multiple options={members} value={selected} getOptionLabel={label}
                                  isOptionEqualToValue={(a, b) => a.id === b.id}
                                  onChange={(_e, v) => {
                                      const ids = v.map((m) => m.id);
                                      setMemberIds(ids);
                                      if (payerId && !ids.includes(payerId)) setPayerId('');
                                      if (!name.trim() && v.length > 0) setName(t('households.defaultName', { name: v[0].lastName }));
                                  }}
                                  renderInput={(p) => <TextField {...p} required label={t('households.columns.members')} />} />
                    <TextField select label={t('households.columns.payer')} value={payerId}
                               helperText={t('households.payerHelp')} onChange={(e) => setPayerId(e.target.value)}>
                        <MenuItem value="">{t('households.noPayer')}</MenuItem>
                        {selected.map((m) => <MenuItem key={m.id} value={m.id}>{label(m)}</MenuItem>)}
                    </TextField>
                    <Box>
                        <FormControlLabel control={<Switch checked={shareAddress} onChange={(e) => setShareAddress(e.target.checked)} />}
                                          label={t('households.shareAddress')} />
                        <FormControlLabel control={<Switch checked={shareBank} onChange={(e) => setShareBank(e.target.checked)} />}
                                          label={t('households.shareBank')} />
                    </Box>
                    {(shareAddress || shareBank) && (
                        <Alert severity="info">{t('households.shareHelp')}</Alert>
                    )}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" disabled={!name.trim() || memberIds.length === 0}
                        onClick={() => onSave({ name: name.trim(), payerId: payerId || undefined, shareAddress, shareBank, memberIds })}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
