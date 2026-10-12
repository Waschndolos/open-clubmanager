import { useCallback, useEffect, useState } from 'react';
import {
    Autocomplete,
    Box,
    Button,
    Checkbox,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControlLabel,
    IconButton,
    Tab,
    Tabs,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { Add, Delete, Edit, WorkspacePremiumOutlined } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { useNotification } from '../../context/NotificationContext';
import { fetchMembers } from '../../api/members';
import { isoDate } from '../../api/honors';
import { Member } from '../../api/types';
import {
    createQualification,
    deleteQualification,
    deleteQualificationType,
    fetchQualifications,
    fetchQualificationTypes,
    Qualification,
    QualificationStatus,
    QualificationType,
    saveQualificationType,
} from '../../api/qualifications';

const STATUS_COLOR: Record<QualificationStatus, 'success' | 'warning' | 'error'> = {
    valid: 'success', expiring: 'warning', expired: 'error',
};

export default function Qualifications() {
    const { t, i18n } = useTranslation();
    const { addNotification } = useNotification();
    const [tab, setTab] = useState(0);
    const [list, setList] = useState<Qualification[]>([]);
    const [types, setTypes] = useState<QualificationType[]>([]);
    const [forbidden, setForbidden] = useState(false);
    const [addOpen, setAddOpen] = useState(false);
    const [editType, setEditType] = useState<Partial<QualificationType> | null>(null);

    const reload = useCallback(async () => {
        try {
            const [q, ty] = await Promise.all([fetchQualifications(), fetchQualificationTypes()]);
            setList(q);
            setTypes(ty);
            setForbidden(false);
        } catch (e) {
            if ((e as { response?: { status?: number } }).response?.status === 403) setForbidden(true);
            else addNotification(t('qualifications.loadFailed'));
        }
    }, [addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const run = async (fn: () => Promise<unknown>) => {
        try {
            await fn();
            await reload();
        } catch {
            addNotification(t('qualifications.saveFailed'));
        }
    };

    const fmt = (iso?: string) => (iso ? new Date(`${iso}T00:00:00`).toLocaleDateString(i18n.language) : '–');

    const cols: GridColDef<Qualification>[] = [
        {
            field: 'status', headerName: t('qualifications.columns.status'), width: 150,
            renderCell: ({ row }) => <Chip size="small" color={STATUS_COLOR[row.status]} label={t(`qualifications.status.${row.status}`)} />,
        },
        { field: 'member', headerName: t('qualifications.columns.member'), flex: 1, minWidth: 180, valueGetter: (_v, r) => `${r.member.firstName} ${r.member.lastName} (${r.member.number})` },
        { field: 'type', headerName: t('qualifications.columns.type'), flex: 1, minWidth: 160, valueGetter: (_v, r) => r.type.name },
        {
            field: 'issuedDate', headerName: t('qualifications.columns.issued'), width: 140,
            valueGetter: (_v, r) => fmt(r.issuedDate),
        },
        { field: 'expiryDate', headerName: t('qualifications.columns.expiry'), width: 130, valueGetter: (_v, r) => fmt(r.expiryDate) },
        { field: 'note', headerName: t('qualifications.columns.note'), flex: 1, minWidth: 140 },
        {
            field: 'actions', headerName: '', width: 70, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <IconButton aria-label={t('qualifications.delete')} onClick={() => {
                    if (window.confirm(t('qualifications.confirmDelete'))) void run(() => deleteQualification(row.id));
                }}><Delete /></IconButton>
            ),
        },
    ];

    const typeCols: GridColDef<QualificationType>[] = [
        { field: 'name', headerName: t('qualifications.columns.type'), flex: 1, minWidth: 200 },
        {
            field: 'validityMonths', headerName: t('qualifications.validity'), width: 170,
            valueGetter: (_v, r) => (r.validityMonths ? t('qualifications.months', { count: r.validityMonths }) : t('qualifications.unlimited')),
        },
        {
            field: 'inspectionOnly', headerName: t('qualifications.inspectionOnly'), width: 200,
            valueGetter: (_v, r) => (r.inspectionOnly ? t('qualifications.yes') : ''),
        },
        {
            field: 'actions', headerName: '', width: 110, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <>
                    <IconButton aria-label={t('qualifications.editType')} onClick={() => setEditType(row)}><Edit /></IconButton>
                    <IconButton aria-label={t('qualifications.deleteType')} onClick={() => {
                        if (window.confirm(t('qualifications.confirmDeleteType'))) void run(() => deleteQualificationType(row.id));
                    }}><Delete /></IconButton>
                </>
            ),
        },
    ];

    return (
        <Box p={3}>
            <PageHeader
                title={t('qualifications.title')}
                subtitle={t('qualifications.subtitle')}
                icon={<WorkspacePremiumOutlined fontSize="small" />}
                actions={
                    !forbidden && (
                        <>
                            <Button variant="outlined" onClick={() => setEditType({ inspectionOnly: false })}>{t('qualifications.addType')}</Button>
                            <Button variant="contained" startIcon={<Add />} onClick={() => setAddOpen(true)} disabled={types.length === 0}>
                                {t('qualifications.add')}
                            </Button>
                        </>
                    )
                }
            />
            {forbidden ? (
                <Typography>{t('qualifications.forbidden')}</Typography>
            ) : (
                <>
                    <Tabs value={tab} onChange={(_e, v) => setTab(v)} sx={{ mb: 2 }}>
                        <Tab label={`${t('qualifications.tabs.overview')} (${list.length})`} />
                        <Tab label={`${t('qualifications.tabs.types')} (${types.length})`} />
                    </Tabs>
                    {tab === 0 ? (
                        <DataGrid rows={list} columns={cols} autoHeight disableRowSelectionOnClick
                                  localeText={{ noRowsLabel: t('qualifications.empty') }} />
                    ) : (
                        <DataGrid rows={types} columns={typeCols} autoHeight disableRowSelectionOnClick
                                  localeText={{ noRowsLabel: t('qualifications.emptyTypes') }} />
                    )}
                </>
            )}
            {addOpen && (
                <AddDialog types={types} onClose={() => setAddOpen(false)} onSave={async (memberId, data) => {
                    setAddOpen(false);
                    await run(() => createQualification(memberId, data));
                }} />
            )}
            {editType && (
                <TypeDialog type={editType} onClose={() => setEditType(null)} onSave={async (ty) => {
                    setEditType(null);
                    await run(() => saveQualificationType(ty));
                }} />
            )}
        </Box>
    );
}

function AddDialog({ types, onClose, onSave }: {
    types: QualificationType[];
    onClose: () => void;
    onSave: (memberId: string, data: { typeId: string; issuedDate: string; expiryDate?: string; note?: string }) => Promise<void>;
}) {
    const { t } = useTranslation();
    const [members, setMembers] = useState<Member[]>([]);
    const [member, setMember] = useState<Member | null>(null);
    const [type, setType] = useState<QualificationType | null>(null);
    const [issued, setIssued] = useState(isoDate(new Date()));
    const [expiry, setExpiry] = useState('');
    const [note, setNote] = useState('');

    useEffect(() => { fetchMembers().then(setMembers).catch(() => undefined); }, []);

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{t('qualifications.add')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <Autocomplete options={members} value={member} onChange={(_e, v) => setMember(v)}
                                  getOptionLabel={(m) => `${m.firstName} ${m.lastName} (${m.number})`}
                                  renderInput={(p) => <TextField {...p} label={t('qualifications.columns.member')} autoFocus />} />
                    <Autocomplete options={types} value={type} onChange={(_e, v) => setType(v)} getOptionLabel={(ty) => ty.name}
                                  renderInput={(p) => <TextField {...p} label={t('qualifications.columns.type')} />} />
                    <TextField type="date" value={issued} onChange={(e) => setIssued(e.target.value)} slotProps={{ inputLabel: { shrink: true } }}
                               label={type?.inspectionOnly ? t('qualifications.inspectedOn') : t('qualifications.columns.issued')} />
                    <TextField type="date" value={expiry} onChange={(e) => setExpiry(e.target.value)} slotProps={{ inputLabel: { shrink: true } }}
                               label={t('qualifications.columns.expiry')}
                               helperText={type?.validityMonths ? t('qualifications.expiryHint', { count: type.validityMonths }) : undefined} />
                    {type?.inspectionOnly ? (
                        <Typography variant="body2" color="text.secondary">{t('qualifications.inspectionHint')}</Typography>
                    ) : (
                        <TextField multiline minRows={2} label={t('qualifications.columns.note')} value={note} onChange={(e) => setNote(e.target.value)} />
                    )}
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" disabled={!member || !type || !issued}
                        onClick={() => member && type && onSave(member.id, {
                            typeId: type.id, issuedDate: issued, expiryDate: expiry || undefined,
                            note: type.inspectionOnly ? undefined : note.trim() || undefined,
                        })}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}

function TypeDialog({ type, onClose, onSave }: {
    type: Partial<QualificationType>;
    onClose: () => void;
    onSave: (t: Partial<QualificationType> & { name: string }) => Promise<void>;
}) {
    const { t } = useTranslation();
    const [name, setName] = useState(type.name ?? '');
    const [months, setMonths] = useState(type.validityMonths ? String(type.validityMonths) : '');
    const [inspectionOnly, setInspectionOnly] = useState(!!type.inspectionOnly);
    const m = months === '' ? undefined : parseInt(months, 10);

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="xs">
            <DialogTitle>{type.id ? t('qualifications.editType') : t('qualifications.addType')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField autoFocus required label={t('qualifications.columns.type')} value={name} onChange={(e) => setName(e.target.value)} />
                    <TextField type="number" label={t('qualifications.validity')} helperText={t('qualifications.validityHint')}
                               value={months} onChange={(e) => setMonths(e.target.value)} slotProps={{ htmlInput: { min: 1, max: 1200 } }} />
                    <Tooltip title={t('qualifications.inspectionHint')}>
                        <FormControlLabel label={t('qualifications.inspectionOnly')}
                                          control={<Checkbox checked={inspectionOnly} onChange={(e) => setInspectionOnly(e.target.checked)} />} />
                    </Tooltip>
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" disabled={!name.trim() || (m !== undefined && !(m >= 1 && m <= 1200))}
                        onClick={() => onSave({ ...type, name: name.trim(), validityMonths: m, inspectionOnly })}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
