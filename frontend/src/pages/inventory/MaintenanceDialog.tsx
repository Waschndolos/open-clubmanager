import { useCallback, useEffect, useState } from 'react';
import {
    Box,
    Button,
    Checkbox,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    Divider,
    FormControlLabel,
    Grid,
    IconButton,
    List,
    ListItem,
    ListItemText,
    MenuItem,
    TextField,
    Typography,
} from '@mui/material';
import { Delete } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { useNotification } from '../../context/NotificationContext';
import { InventoryItem } from '../../api/types';
import {
    CONDITIONS,
    deleteMaintenance,
    fetchItemMaintenance,
    ItemCondition,
    Maintenance,
    MaintenanceState,
    recordMaintenance,
    setItemStatus,
} from '../../api/maintenance';

type Props = {
    item: InventoryItem;
    state?: MaintenanceState;
    canEdit: boolean;
    /** May book the cost as an expense (treasurer with cash book access). */
    canBook: boolean;
    onClose: () => void;
    onChanged: () => void;
};

const today = () => new Date().toISOString().slice(0, 10);

export default function MaintenanceDialog({ item, state, canEdit, canBook, onClose, onChanged }: Props) {
    const { t, i18n } = useTranslation();
    const { addNotification } = useNotification();
    const [condition, setCondition] = useState<ItemCondition>(state?.condition ?? 'ready');
    const [interval, setIntervalMonths] = useState(state?.intervalMonths?.toString() ?? '');
    const [nextDue, setNextDue] = useState(state?.nextDue ?? '');
    const [history, setHistory] = useState<Maintenance[]>([]);

    const [date, setDate] = useState(today());
    const [performedBy, setPerformedBy] = useState('');
    const [note, setNote] = useState('');
    const [cost, setCost] = useState('');
    const [after, setAfter] = useState<ItemCondition | ''>(state && state.condition !== 'ready' ? 'ready' : '');
    const [book, setBook] = useState(false);

    const reload = useCallback(() => {
        fetchItemMaintenance(item.id).then(setHistory).catch(() => addNotification(t('inventory.maintenance.loadFailed')));
    }, [item.id, addNotification, t]);
    useEffect(reload, [reload]);

    const fail = () => addNotification(t('inventory.maintenance.saveFailed'));
    const intervalValid = interval === '' || (Number.isInteger(Number(interval)) && Number(interval) > 0);
    const costValue = cost === '' ? undefined : Number(cost);
    const costValid = costValue === undefined || (Number.isFinite(costValue) && costValue >= 0);
    const formatDate = (d?: string) => d ? new Date(`${d}T00:00:00`).toLocaleDateString(i18n.language) : '–';

    const saveStatus = () => {
        setItemStatus(item.id, {
            condition, intervalMonths: interval ? Number(interval) : undefined, nextDue: nextDue || undefined,
        }).then(onChanged).catch(fail);
    };

    const record = () => {
        recordMaintenance(item.id, {
            date, performedBy: performedBy || undefined, note: note || undefined, cost: costValue,
            condition: after || undefined,
            bookExpense: book && costValue ? {
                description: t('inventory.maintenance.expenseDescription', { name: item.name }),
                category: t('inventory.maintenance.expenseCategory'),
            } : undefined,
        }).then(() => {
            setPerformedBy('');
            setNote('');
            setCost('');
            setBook(false);
            if (book && costValue) addNotification(t('inventory.maintenance.expenseBooked'));
            reload();
            onChanged();
        }).catch(fail);
    };

    // Recording a maintenance moves the due date and may change the condition;
    // keep the status form in sync with the saved state.
    useEffect(() => {
        if (!state) return;
        setCondition(state.condition);
        setIntervalMonths(state.intervalMonths?.toString() ?? '');
        setNextDue(state.nextDue ?? '');
    }, [state]);

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="md">
            <DialogTitle>{t('inventory.maintenance.title', { name: item.name })}</DialogTitle>
            <DialogContent>
                <Typography variant="subtitle2" sx={{ mt: 1, mb: 1 }}>{t('inventory.maintenance.status')}</Typography>
                <Box component="fieldset" disabled={!canEdit} sx={{ border: 0, p: 0, m: 0, minWidth: 0 }}>
                    <Grid container spacing={2} alignItems="center">
                        <Grid size={{ xs: 12, sm: 4 }}>
                            <TextField select fullWidth size="small" label={t('inventory.maintenance.condition')} value={condition}
                                       disabled={!canEdit} onChange={(e) => setCondition(e.target.value as ItemCondition)}>
                                {CONDITIONS.map((c) => <MenuItem key={c} value={c}>{t(`inventory.condition.${c}`)}</MenuItem>)}
                            </TextField>
                        </Grid>
                        <Grid size={{ xs: 6, sm: 3 }}>
                            <TextField fullWidth size="small" type="number" label={t('inventory.maintenance.interval')}
                                       value={interval} error={!intervalValid} onChange={(e) => setIntervalMonths(e.target.value)}
                                       slotProps={{ htmlInput: { min: 1 } }} />
                        </Grid>
                        <Grid size={{ xs: 6, sm: 3 }}>
                            <TextField fullWidth size="small" type="date" label={t('inventory.maintenance.nextDue')} value={nextDue}
                                       onChange={(e) => setNextDue(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                        </Grid>
                        <Grid size={{ xs: 12, sm: 2 }}>
                            {canEdit && <Button fullWidth variant="outlined" disabled={!intervalValid} onClick={saveStatus}>{t('buttons.save')}</Button>}
                        </Grid>
                    </Grid>
                </Box>
                {condition !== 'ready' && (
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>{t('inventory.maintenance.notLendable')}</Typography>
                )}

                {canEdit && <>
                    <Divider sx={{ my: 3 }} />
                    <Typography variant="subtitle2" sx={{ mb: 1 }}>{t('inventory.maintenance.record')}</Typography>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12, sm: 4 }}>
                            <TextField fullWidth size="small" type="date" required label={t('inventory.maintenance.date')} value={date}
                                       onChange={(e) => setDate(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
                        </Grid>
                        <Grid size={{ xs: 12, sm: 4 }}>
                            <TextField fullWidth size="small" label={t('inventory.maintenance.performedBy')} value={performedBy}
                                       onChange={(e) => setPerformedBy(e.target.value)} />
                        </Grid>
                        <Grid size={{ xs: 12, sm: 4 }}>
                            <TextField fullWidth size="small" type="number" label={t('inventory.maintenance.cost')} value={cost}
                                       error={!costValid} onChange={(e) => setCost(e.target.value)} slotProps={{ htmlInput: { min: 0, step: '0.01' } }} />
                        </Grid>
                        <Grid size={{ xs: 12, sm: 8 }}>
                            <TextField fullWidth size="small" label={t('inventory.maintenance.note')} value={note}
                                       onChange={(e) => setNote(e.target.value)} />
                        </Grid>
                        <Grid size={{ xs: 12, sm: 4 }}>
                            <TextField select fullWidth size="small" label={t('inventory.maintenance.conditionAfter')} value={after}
                                       onChange={(e) => setAfter(e.target.value as ItemCondition | '')}>
                                <MenuItem value="">{t('inventory.maintenance.unchanged')}</MenuItem>
                                {CONDITIONS.map((c) => <MenuItem key={c} value={c}>{t(`inventory.condition.${c}`)}</MenuItem>)}
                            </TextField>
                        </Grid>
                        <Grid size={{ xs: 12, sm: 8 }}>
                            {canBook && (
                                <FormControlLabel control={<Checkbox checked={book} disabled={!costValue} onChange={(e) => setBook(e.target.checked)} />}
                                                  label={t('inventory.maintenance.bookExpense')} />
                            )}
                        </Grid>
                        <Grid size={{ xs: 12, sm: 4 }} display="flex" justifyContent="flex-end" alignItems="center">
                            <Button variant="contained" disabled={!date || !costValid} onClick={record}>{t('inventory.maintenance.add')}</Button>
                        </Grid>
                    </Grid>
                    {state?.intervalMonths && (
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {t('inventory.maintenance.movesDue', { count: state.intervalMonths })}
                        </Typography>
                    )}
                </>}

                <Divider sx={{ my: 3 }} />
                <Typography variant="subtitle2">{t('inventory.maintenance.history')}</Typography>
                {history.length === 0 && (
                    <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>{t('inventory.maintenance.noHistory')}</Typography>
                )}
                <List dense>
                    {history.map((m) => (
                        <ListItem key={m.id} disableGutters secondaryAction={canEdit && (
                            <IconButton edge="end" aria-label={t('tooltips.delete')} onClick={() => {
                                if (window.confirm(t('inventory.maintenance.confirmDelete'))) {
                                    deleteMaintenance(m.id).then(() => { reload(); onChanged(); }).catch(fail);
                                }
                            }}><Delete fontSize="small" /></IconButton>
                        )}>
                            <ListItemText
                                primary={[formatDate(m.date), m.performedBy, m.note].filter(Boolean).join(' · ')}
                                secondary={m.cost !== undefined
                                    ? `${m.cost.toLocaleString(i18n.language, { style: 'currency', currency: 'EUR' })}${m.transactionId ? ` · ${t('inventory.maintenance.booked')}` : ''}`
                                    : undefined} />
                        </ListItem>
                    ))}
                </List>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.close')}</Button>
            </DialogActions>
        </Dialog>
    );
}
