import React, { useEffect, useState } from 'react';
import {
    Alert,
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    FormControl,
    IconButton,
    InputLabel,
    MenuItem,
    Paper,
    Select,
    Snackbar,
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
import { Delete, Edit, PersonAdd } from '@mui/icons-material';
import PeopleIcon from '@mui/icons-material/People';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { AppRole, AppUser } from '../../api/types';
import { createAppUser, deleteAppUser, listAppUsers, updateAppUser } from '../../api/users';

const APP_ROLES: AppRole[] = ['ADMIN', 'TREASURER', 'SECRETARY', 'READONLY'];

interface UserFormState {
    email: string;
    password: string;
    appRole: AppRole;
}

const defaultForm: UserFormState = { email: '', password: '', appRole: 'READONLY' };

export default function Users() {
    const { t } = useTranslation();
    const [users, setUsers] = useState<AppUser[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const [createOpen, setCreateOpen] = useState(false);
    const [editUser, setEditUser] = useState<AppUser | null>(null);
    const [deleteUser, setDeleteUser] = useState<AppUser | null>(null);

    const [form, setForm] = useState<UserFormState>(defaultForm);
    const [editRole, setEditRole] = useState<AppRole>('READONLY');

    const [snack, setSnack] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
        open: false,
        message: '',
        severity: 'success',
    });

    const load = async () => {
        try {
            setLoading(true);
            const data = await listAppUsers();
            setUsers(data);
            setError(null);
        } catch {
            setError(t('users.errors.loadFailed'));
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, []);

    const handleCreate = async () => {
        try {
            await createAppUser(form.email, form.password, form.appRole);
            setCreateOpen(false);
            setForm(defaultForm);
            await load();
            setSnack({ open: true, message: t('users.snack.created'), severity: 'success' });
        } catch {
            setSnack({ open: true, message: t('users.errors.createFailed'), severity: 'error' });
        }
    };

    const handleUpdate = async () => {
        if (!editUser) return;
        try {
            await updateAppUser(editUser.id, undefined, editRole);
            setEditUser(null);
            await load();
            setSnack({ open: true, message: t('users.snack.updated'), severity: 'success' });
        } catch {
            setSnack({ open: true, message: t('users.errors.updateFailed'), severity: 'error' });
        }
    };

    const handleDelete = async () => {
        if (!deleteUser) return;
        try {
            await deleteAppUser(deleteUser.id);
            setDeleteUser(null);
            await load();
            setSnack({ open: true, message: t('users.snack.deleted'), severity: 'success' });
        } catch {
            setSnack({ open: true, message: t('users.errors.deleteFailed'), severity: 'error' });
        }
    };

    const roleColor = (role: AppRole): 'error' | 'warning' | 'info' | 'default' => {
        switch (role) {
            case 'ADMIN': return 'error';
            case 'TREASURER': return 'warning';
            case 'SECRETARY': return 'info';
            default: return 'default';
        }
    };

    return (
        <Box sx={{ p: 4 }}>
            <PageHeader
                title={t('users.title')}
                icon={<PeopleIcon fontSize="small" />}
                actions={
                    <Button variant="contained" startIcon={<PersonAdd />} onClick={() => setCreateOpen(true)}>
                        {t('users.createButton')}
                    </Button>
                }
            />

            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

            {!loading && users.length === 0 && (
                <Typography color="text.secondary">{t('users.noUsers')}</Typography>
            )}

            {users.length > 0 && (
                <TableContainer component={Paper}>
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('users.columns.email')}</TableCell>
                                <TableCell>{t('users.columns.role')}</TableCell>
                                <TableCell>{t('users.columns.createdAt')}</TableCell>
                                <TableCell align="right">{t('users.columns.actions')}</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {users.map((u) => (
                                <TableRow key={u.id} hover>
                                    <TableCell>{u.email}</TableCell>
                                    <TableCell>
                                        <Chip
                                            label={t(`users.roles.${u.appRole}`)}
                                            size="small"
                                            color={roleColor(u.appRole)}
                                        />
                                    </TableCell>
                                    <TableCell>
                                        {new Date(u.createdAt).toLocaleDateString()}
                                    </TableCell>
                                    <TableCell align="right">
                                        <Tooltip title={t('tooltips.edit')}>
                                            <IconButton size="small" onClick={() => { setEditUser(u); setEditRole(u.appRole); }}>
                                                <Edit fontSize="small" />
                                            </IconButton>
                                        </Tooltip>
                                        <Tooltip title={t('tooltips.delete')}>
                                            <IconButton size="small" color="error" onClick={() => setDeleteUser(u)}>
                                                <Delete fontSize="small" />
                                            </IconButton>
                                        </Tooltip>
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}

            {/* Create dialog */}
            <Dialog open={createOpen} onClose={() => setCreateOpen(false)} maxWidth="xs" fullWidth>
                <DialogTitle>{t('users.dialogs.create.title')}</DialogTitle>
                <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: '16px !important' }}>
                    <TextField
                        label={t('users.dialogs.create.email')}
                        type="email"
                        value={form.email}
                        onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                        fullWidth
                        required
                    />
                    <TextField
                        label={t('users.dialogs.create.password')}
                        type="password"
                        value={form.password}
                        onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
                        fullWidth
                        required
                    />
                    <FormControl fullWidth required>
                        <InputLabel>{t('users.dialogs.create.role')}</InputLabel>
                        <Select
                            value={form.appRole}
                            label={t('users.dialogs.create.role')}
                            onChange={(e) => setForm((f) => ({ ...f, appRole: e.target.value as AppRole }))}
                        >
                            {APP_ROLES.map((r) => (
                                <MenuItem key={r} value={r}>{t(`users.roles.${r}`)}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setCreateOpen(false)}>{t('buttons.abort')}</Button>
                    <Button
                        variant="contained"
                        onClick={handleCreate}
                        disabled={!form.email || !form.password}
                    >
                        {t('buttons.save')}
                    </Button>
                </DialogActions>
            </Dialog>

            {/* Edit dialog */}
            <Dialog open={!!editUser} onClose={() => setEditUser(null)} maxWidth="xs" fullWidth>
                <DialogTitle>{t('users.dialogs.edit.title')}</DialogTitle>
                <DialogContent sx={{ pt: '16px !important' }}>
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                        {editUser?.email}
                    </Typography>
                    <FormControl fullWidth>
                        <InputLabel>{t('users.dialogs.edit.role')}</InputLabel>
                        <Select
                            value={editRole}
                            label={t('users.dialogs.edit.role')}
                            onChange={(e) => setEditRole(e.target.value as AppRole)}
                        >
                            {APP_ROLES.map((r) => (
                                <MenuItem key={r} value={r}>{t(`users.roles.${r}`)}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setEditUser(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" onClick={handleUpdate}>{t('buttons.save')}</Button>
                </DialogActions>
            </Dialog>

            {/* Delete confirmation dialog */}
            <Dialog open={!!deleteUser} onClose={() => setDeleteUser(null)} maxWidth="xs" fullWidth>
                <DialogTitle>{t('users.dialogs.delete.title')}</DialogTitle>
                <DialogContent>
                    <Typography>
                        {t('users.dialogs.delete.confirm', { email: deleteUser?.email })}
                    </Typography>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setDeleteUser(null)}>{t('buttons.abort')}</Button>
                    <Button variant="contained" color="error" onClick={handleDelete}>
                        {t('buttons.delete')}
                    </Button>
                </DialogActions>
            </Dialog>

            <Snackbar
                open={snack.open}
                autoHideDuration={4000}
                onClose={() => setSnack((s) => ({ ...s, open: false }))}
            >
                <Alert severity={snack.severity} onClose={() => setSnack((s) => ({ ...s, open: false }))}>
                    {snack.message}
                </Alert>
            </Snackbar>
        </Box>
    );
}
