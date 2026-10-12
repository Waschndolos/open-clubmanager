import { useCallback, useEffect, useState } from 'react';
import {
    Autocomplete,
    Box,
    Button,
    Chip,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    MenuItem,
    Select,
    TextField,
    ToggleButton,
    ToggleButtonGroup,
} from '@mui/material';
import { Add, Delete, Edit, TaskAltOutlined } from '@mui/icons-material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useTranslation } from 'react-i18next';
import PageHeader from '../../components/common/PageHeader';
import { useAuth } from '../../context/AuthContext';
import { usePermission } from '../../context/CollaborationContext';
import { useNotification } from '../../context/NotificationContext';
import { fetchMembers } from '../../api/members';
import { fetchRoles } from '../../api/roles';
import { Member, Role } from '../../api/types';
import {
    createTask,
    deleteTask,
    fetchTasks,
    Task,
    TaskFilter,
    TaskInput,
    TaskRecurrence,
    TaskStatus,
    taskToInput,
    updateTask,
} from '../../api/tasks';

const STATUSES: TaskStatus[] = ['open', 'in_progress', 'done'];
const RECURRENCES: TaskRecurrence[] = ['none', 'weekly', 'monthly', 'yearly'];
const FILTERS: TaskFilter[] = ['all', 'mine', 'overdue', 'done'];

export default function Tasks() {
    const { t, i18n } = useTranslation();
    const { appRole } = useAuth();
    const { addNotification } = useNotification();
    const area = usePermission('membersCore');
    const canEdit = appRole !== 'READONLY' && area === 'write';

    const [filter, setFilter] = useState<TaskFilter>('all');
    const [tasks, setTasks] = useState<Task[]>([]);
    const [editing, setEditing] = useState<Partial<Task> | null>(null);

    const reload = useCallback(async () => {
        try {
            setTasks(await fetchTasks(filter));
        } catch {
            addNotification(t('tasks.loadFailed'));
        }
    }, [filter, addNotification, t]);

    useEffect(() => { void reload(); }, [reload]);

    const run = async (fn: () => Promise<unknown>) => {
        try {
            await fn();
            await reload();
        } catch {
            addNotification(t('tasks.saveFailed'));
            await reload();
        }
    };

    const cols: GridColDef<Task>[] = [
        {
            field: 'title', headerName: t('tasks.columns.title'), flex: 1, minWidth: 220,
            renderCell: ({ row }) => (
                <Box sx={{ textDecoration: row.status === 'done' ? 'line-through' : 'none' }}>{row.title}</Box>
            ),
        },
        {
            field: 'status', headerName: t('tasks.columns.status'), width: 160,
            renderCell: ({ row }) => canEdit ? (
                <Select size="small" variant="standard" value={row.status} disableUnderline
                        onChange={(e) => run(() => updateTask(row.id, { ...taskToInput(row), status: e.target.value as TaskStatus }))}>
                    {STATUSES.map((s) => <MenuItem key={s} value={s}>{t(`tasks.status.${s}`)}</MenuItem>)}
                </Select>
            ) : t(`tasks.status.${row.status}`),
        },
        {
            field: 'dueDate', headerName: t('tasks.columns.due'), width: 170,
            renderCell: ({ row }) => (
                <>
                    {row.dueDate ? new Date(`${row.dueDate}T00:00:00`).toLocaleDateString(i18n.language) : '–'}
                    {row.overdue && <Chip size="small" color="error" label={t('tasks.overdue')} sx={{ ml: 1 }} />}
                </>
            ),
        },
        {
            field: 'assignee', headerName: t('tasks.columns.assignee'), flex: 1, minWidth: 170,
            valueGetter: (_v, r) => r.assigneeMember ? `${r.assigneeMember.firstName} ${r.assigneeMember.lastName}` : (r.assigneeRole?.name ?? '–'),
        },
        {
            field: 'recurrence', headerName: t('tasks.columns.recurrence'), width: 130,
            valueGetter: (_v, r) => r.recurrence === 'none' ? '' : t(`tasks.recurrence.${r.recurrence}`),
        },
        ...(canEdit ? [{
            field: 'actions', headerName: '', width: 110, sortable: false, filterable: false,
            renderCell: ({ row }) => (
                <>
                    <IconButton aria-label={t('tasks.edit')} onClick={() => setEditing(row)}><Edit /></IconButton>
                    <IconButton aria-label={t('tasks.delete')} onClick={() => {
                        if (window.confirm(t('tasks.confirmDelete'))) void run(() => deleteTask(row.id));
                    }}><Delete /></IconButton>
                </>
            ),
        } as GridColDef<Task>] : []),
    ];

    return (
        <Box p={3}>
            <PageHeader
                title={t('tasks.title')}
                subtitle={t('tasks.subtitle')}
                icon={<TaskAltOutlined fontSize="small" />}
                actions={canEdit && (
                    <Button variant="contained" startIcon={<Add />} onClick={() => setEditing({ status: 'open', recurrence: 'none' })}>
                        {t('tasks.add')}
                    </Button>
                )}
            />
            <ToggleButtonGroup exclusive size="small" value={filter} sx={{ mb: 2 }} aria-label={t('tasks.filter.label')}
                               onChange={(_e, v: TaskFilter | null) => v && setFilter(v)}>
                {FILTERS.map((f) => <ToggleButton key={f} value={f}>{t(`tasks.filter.${f}`)}</ToggleButton>)}
            </ToggleButtonGroup>
            <DataGrid rows={tasks} columns={cols} autoHeight disableRowSelectionOnClick
                      localeText={{ noRowsLabel: t('tasks.empty') }} />
            {editing && (
                <TaskDialog task={editing} onClose={() => setEditing(null)} onSave={async (input) => {
                    const id = editing.id;
                    setEditing(null);
                    await run(() => (id ? updateTask(id, input) : createTask(input)));
                }} />
            )}
        </Box>
    );
}

function TaskDialog({ task, onClose, onSave }: { task: Partial<Task>; onClose: () => void; onSave: (t: TaskInput) => Promise<void> }) {
    const { t } = useTranslation();
    const [members, setMembers] = useState<Member[]>([]);
    const [roles, setRoles] = useState<Role[]>([]);
    const [title, setTitle] = useState(task.title ?? '');
    const [description, setDescription] = useState(task.description ?? '');
    const [due, setDue] = useState(task.dueDate ?? '');
    const [status, setStatus] = useState<TaskStatus>(task.status ?? 'open');
    const [recurrence, setRecurrence] = useState<TaskRecurrence>(task.recurrence ?? 'none');
    const [assigneeType, setAssigneeType] = useState<'none' | 'member' | 'role'>(
        task.assigneeMember ? 'member' : task.assigneeRole ? 'role' : 'none');
    const [memberId, setMemberId] = useState(task.assigneeMember?.id ?? '');
    const [roleId, setRoleId] = useState(task.assigneeRole?.id ?? '');
    const [linkedId, setLinkedId] = useState(task.linkedMember?.id ?? '');

    useEffect(() => {
        fetchMembers().then(setMembers).catch(() => undefined);
        fetchRoles().then(setRoles).catch(() => undefined);
    }, []);

    const memberLabel = (m: Member) => `${m.firstName} ${m.lastName} (${m.number})`;
    const recurrenceNeedsDue = recurrence !== 'none' && !due;

    return (
        <Dialog open onClose={onClose} fullWidth maxWidth="sm">
            <DialogTitle>{task.id ? t('tasks.edit') : t('tasks.add')}</DialogTitle>
            <DialogContent>
                <Box display="flex" flexDirection="column" gap={2} mt={1}>
                    <TextField autoFocus required label={t('tasks.columns.title')} value={title} onChange={(e) => setTitle(e.target.value)} />
                    <TextField multiline minRows={2} label={t('tasks.description')} value={description} onChange={(e) => setDescription(e.target.value)} />
                    <Box display="flex" gap={2}>
                        <TextField type="date" label={t('tasks.columns.due')} value={due} onChange={(e) => setDue(e.target.value)}
                                   slotProps={{ inputLabel: { shrink: true } }} sx={{ flex: 1 }} />
                        <TextField select label={t('tasks.columns.status')} value={status} onChange={(e) => setStatus(e.target.value as TaskStatus)} sx={{ flex: 1 }}>
                            {STATUSES.map((s) => <MenuItem key={s} value={s}>{t(`tasks.status.${s}`)}</MenuItem>)}
                        </TextField>
                    </Box>
                    <TextField select label={t('tasks.columns.recurrence')} value={recurrence} error={recurrenceNeedsDue}
                               helperText={recurrenceNeedsDue ? t('tasks.recurrenceNeedsDue') : undefined}
                               onChange={(e) => setRecurrence(e.target.value as TaskRecurrence)}>
                        {RECURRENCES.map((r) => <MenuItem key={r} value={r}>{r === 'none' ? t('tasks.recurrence.none') : t(`tasks.recurrence.${r}`)}</MenuItem>)}
                    </TextField>
                    <TextField select label={t('tasks.assignTo')} value={assigneeType} onChange={(e) => setAssigneeType(e.target.value as 'none' | 'member' | 'role')}>
                        <MenuItem value="none">{t('tasks.assignNone')}</MenuItem>
                        <MenuItem value="member">{t('tasks.assignMember')}</MenuItem>
                        <MenuItem value="role">{t('tasks.assignRole')}</MenuItem>
                    </TextField>
                    {assigneeType === 'member' && (
                        <Autocomplete options={members} value={members.find((m) => m.id === memberId) ?? null}
                                      onChange={(_e, v) => setMemberId(v?.id ?? '')} getOptionLabel={memberLabel}
                                      renderInput={(p) => <TextField {...p} label={t('tasks.assignMember')} />} />
                    )}
                    {assigneeType === 'role' && (
                        <Autocomplete options={roles} value={roles.find((r) => r.id === roleId) ?? null}
                                      onChange={(_e, v) => setRoleId(v?.id ?? '')} getOptionLabel={(r) => r.name}
                                      renderInput={(p) => <TextField {...p} label={t('tasks.assignRole')} />} />
                    )}
                    <Autocomplete options={members} value={members.find((m) => m.id === linkedId) ?? null}
                                  onChange={(_e, v) => setLinkedId(v?.id ?? '')} getOptionLabel={memberLabel}
                                  renderInput={(p) => <TextField {...p} label={t('tasks.linkedMember')} />} />
                </Box>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t('buttons.abort')}</Button>
                <Button variant="contained" disabled={!title.trim() || recurrenceNeedsDue ||
                    (assigneeType === 'member' && !memberId) || (assigneeType === 'role' && !roleId)}
                        onClick={() => onSave({
                            title: title.trim(), description: description.trim() || undefined, dueDate: due || undefined, status, recurrence,
                            assigneeMemberId: assigneeType === 'member' ? memberId : undefined,
                            assigneeRoleId: assigneeType === 'role' ? roleId : undefined,
                            linkedMemberId: linkedId || undefined,
                        })}>
                    {t('buttons.save')}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
