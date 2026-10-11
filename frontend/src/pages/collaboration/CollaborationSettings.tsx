import React, { useCallback, useEffect, useState } from 'react';
import {
    Alert, Box, Button, Chip, CircularProgress, Dialog, DialogActions, DialogContent, DialogTitle, Divider,
    IconButton, Menu, MenuItem, Paper, Snackbar, Stack, Tab, Table, TableBody, TableCell, TableContainer, TableHead,
    TableRow, Tabs, Typography,
} from '@mui/material';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import GroupsIcon from '@mui/icons-material/Groups';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
    AREAS, changeDeviceRole, ClubDevice, ClubInvite, ClubJoinRequest, ClubRole, listDevices, listInvites, listRequests,
    listRoles, revokeDevice, withdrawInvite,
} from '../../api/collaboration';
import PageHeader from '../../components/common/PageHeader';
import ApproveRequestDialog from '../../components/collaboration/ApproveRequestDialog';
import InviteWizard from '../../components/collaboration/InviteWizard';
import { ChoiceCard, PermissionRow, RoleTable, useErrorText, useRoleName } from '../../components/collaboration/common';
import { useCollaboration } from '../../context/CollaborationContext';

type TabId = 'overview' | 'people' | 'invites' | 'roles' | 'emergency';

function LocalMode() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    return (
        <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="h6">{t('collaboration.settings.localTitle')}</Typography>
            <Typography>{t('collaboration.settings.localText')}</Typography>
            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                <Button variant="contained" onClick={() => navigate('/collaboration/create')}>{t('collaboration.settings.enable')}</Button>
                <Button variant="outlined" onClick={() => navigate('/collaboration/join')}>{t('collaboration.settings.join')}</Button>
                <Button onClick={() => navigate('/collaboration/recover')}>{t('collaboration.setup.recover')}</Button>
            </Box>
        </Paper>
    );
}

function Overview({ roles }: { roles: ClubRole[] }) {
    const { t } = useTranslation();
    const roleName = useRoleName();
    const { status, sync } = useCollaboration();
    if (!status) return null;
    const myRole = roles.find((r) => r.id === status.role);
    const rows: [string, React.ReactNode][] = [
        [t('collaboration.settings.club'), status.clubName],
        [t('collaboration.settings.storage'), <Box component="span" sx={{ wordBreak: 'break-all' }}>{status.storagePath}</Box>],
        [t('collaboration.settings.yourRole'), roleName(status.role)],
        [t('collaboration.settings.device'), [status.person, status.deviceName].filter(Boolean).join(' – ')],
        [t('collaboration.settings.lastSync'), status.lastSync ? new Date(status.lastSync).toLocaleString() : t('collaboration.settings.never')],
    ];
    return (
        <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="h6">{t('collaboration.settings.overviewTitle')}</Typography>
            <Table size="small">
                <TableBody>
                    {rows.map(([k, v]) => (
                        <TableRow key={k}>
                            <TableCell component="th" sx={{ color: 'text.secondary', width: '35%' }}>{k}</TableCell>
                            <TableCell>{v}</TableCell>
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
            {myRole && (
                <Box>
                    <Typography variant="subtitle2" sx={{ mb: 0.5 }}>{t('collaboration.settings.whatYouCan')}</Typography>
                    <PermissionRow permissions={myRole.permissions} />
                </Box>
            )}
            <Box>
                <Button variant="outlined" onClick={sync} disabled={status.syncing}>
                    {status.syncing ? t('collaboration.status.syncing') : t('collaboration.popover.syncNow')}
                </Button>
            </Box>
        </Paper>
    );
}

function People({ roles, onChanged }: { roles: ClubRole[]; onChanged: () => void }) {
    const { t } = useTranslation();
    const roleName = useRoleName();
    const errorText = useErrorText();
    const { status, refresh } = useCollaboration();
    const [devices, setDevices] = useState<ClubDevice[] | null>(null);
    const [requests, setRequests] = useState<ClubJoinRequest[]>([]);
    const [menu, setMenu] = useState<{ anchor: HTMLElement; device: ClubDevice } | null>(null);
    const [removing, setRemoving] = useState<ClubDevice | null>(null);
    const [reason, setReason] = useState<'lost' | 'left' | 'unused'>('lost');
    const [roleFor, setRoleFor] = useState<ClubDevice | null>(null);
    const [newRole, setNewRole] = useState('');
    const [approving, setApproving] = useState<ClubJoinRequest | null>(null);
    const [error, setError] = useState('');
    const [toast, setToast] = useState('');

    const load = useCallback(() => {
        listDevices().then(setDevices, (e) => setError(errorText(e)));
        if (status?.isAdmin) listRequests().then(setRequests, () => {});
    }, [status?.isAdmin]);

    useEffect(load, [load, status?.pendingRequests]);

    const isAdmin = !!status?.isAdmin;
    const deviceLabel = (d: ClubDevice) => d.deviceName || t('collaboration.people.unnamedDevice');
    const persons = (devices ?? []).reduce<Record<string, ClubDevice[]>>((acc, d) => {
        (acc[d.person] ??= []).push(d);
        return acc;
    }, {});

    const doRemove = async () => {
        if (!removing) return;
        try {
            await revokeDevice(removing.deviceId);
            setToast(t('collaboration.people.removed'));
            setRemoving(null);
            load();
            refresh();
            onChanged();
        } catch (e) {
            setError(errorText(e));
            setRemoving(null);
        }
    };

    const lostAreas = (() => {
        if (!roleFor || !newRole) return [];
        const from = roles.find((r) => r.id === roleFor.role)?.permissions;
        const to = roles.find((r) => r.id === newRole)?.permissions;
        if (!from || !to) return [];
        return AREAS.filter((a) => from[a] !== 'none' && to[a] === 'none').map((a) => t(`collaboration.areas.${a}`));
    })();

    const doChangeRole = async () => {
        if (!roleFor) return;
        try {
            await changeDeviceRole(roleFor.deviceId, newRole);
            setToast(t('collaboration.people.roleChanged'));
            setRoleFor(null);
            load();
        } catch (e) {
            setError(errorText(e));
            setRoleFor(null);
        }
    };

    if (devices === null) return <CircularProgress size={24} />;

    return (
        <Stack spacing={2}>
            {error && <Alert severity="error" onClose={() => setError('')}>{error}</Alert>}
            {isAdmin && requests.length > 0 && (
                <Paper sx={{ p: 2 }}>
                    <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1 }}>{t('collaboration.people.pending')}</Typography>
                    {requests.map((r) => (
                        <Box key={r.deviceId} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5, flexWrap: 'wrap' }}>
                            <Typography sx={{ flex: 1, minWidth: 160 }}>
                                {t('collaboration.request.wants', { name: r.person })}{r.deviceName ? ` – ${r.deviceName}` : ''}
                            </Typography>
                            {r.role && <Chip size="small" label={roleName(r.role)} />}
                            {r.error && <Chip size="small" color="warning" label={t(`collaboration.request.invalid.${r.error}`, { defaultValue: t('collaboration.request.invalid.default') })} />}
                            <Button size="small" variant="contained" onClick={() => setApproving(r)}>{t('collaboration.popover.view')}</Button>
                        </Box>
                    ))}
                </Paper>
            )}
            <Paper sx={{ p: 2 }}>
                {Object.entries(persons).map(([person, devs], i) => (
                    <Box key={person}>
                        {i > 0 && <Divider sx={{ my: 1 }} />}
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
                            <Typography sx={{ fontWeight: 600, flex: 1, minWidth: 140 }}>
                                {person} {devs.some((d) => d.me) && <Typography component="span" color="text.secondary">{t('collaboration.people.me')}</Typography>}
                            </Typography>
                            <Chip size="small" label={roleName(devs.find((d) => d.active)?.role ?? devs[0].role)} />
                        </Box>
                        {devs.map((d) => (
                            <Box key={d.deviceId} sx={{ display: 'flex', alignItems: 'center', gap: 1, pl: 2, minHeight: 40 }}>
                                <Typography sx={{ flex: 1, color: d.active ? 'text.primary' : 'text.disabled' }}>{deviceLabel(d)}</Typography>
                                {!d.active && <Typography variant="body2" color="text.disabled">{t('collaboration.people.inactive')}</Typography>}
                                {isAdmin && d.active && !d.me && (
                                    <IconButton size="small" aria-label={`${person} – ${deviceLabel(d)}`} onClick={(e) => setMenu({ anchor: e.currentTarget, device: d })}>
                                        <MoreVertIcon fontSize="small" />
                                    </IconButton>
                                )}
                            </Box>
                        ))}
                    </Box>
                ))}
                {isAdmin && <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>{t('collaboration.people.ownHint')}</Typography>}
            </Paper>

            <Menu open={!!menu} anchorEl={menu?.anchor} onClose={() => setMenu(null)}>
                <MenuItem onClick={() => { setRoleFor(menu!.device); setNewRole(menu!.device.role); setMenu(null); }}>
                    {t('collaboration.people.changeRole')}
                </MenuItem>
                <MenuItem onClick={() => { setRemoving(menu!.device); setReason('lost'); setMenu(null); }}>
                    {t('collaboration.people.removeDevice')}
                </MenuItem>
            </Menu>

            <Dialog open={!!removing} onClose={() => setRemoving(null)} maxWidth="sm" fullWidth>
                {removing && (
                    <>
                        <DialogTitle>{t('collaboration.people.removeTitle', { device: deviceLabel(removing), name: removing.person })}</DialogTitle>
                        <DialogContent>
                            <Stack spacing={1.5}>
                                <Typography sx={{ fontWeight: 600 }}>{t('collaboration.people.why')}</Typography>
                                {(['lost', 'left', 'unused'] as const).map((r) => (
                                    <ChoiceCard key={r} selected={reason === r} onSelect={() => setReason(r)} title={t(`collaboration.people.reasons.${r}`)} />
                                ))}
                                <Box component="ul" sx={{ m: 0, pl: 3 }}>
                                    {[1, 2, 3].map((n) => <li key={n}><Typography>{t(`collaboration.people.consequence${n}`)}</Typography></li>)}
                                </Box>
                                {reason === 'lost' && <Alert severity="warning">{t('collaboration.people.breach')}</Alert>}
                            </Stack>
                        </DialogContent>
                        <DialogActions>
                            <Button onClick={() => setRemoving(null)}>{t('buttons.abort')}</Button>
                            <Button color="error" variant="contained" onClick={doRemove}>{t('collaboration.people.removeDevice')}</Button>
                        </DialogActions>
                    </>
                )}
            </Dialog>

            <Dialog open={!!roleFor} onClose={() => setRoleFor(null)} maxWidth="sm" fullWidth>
                {roleFor && (
                    <>
                        <DialogTitle>{t('collaboration.people.roleTitle', { name: roleFor.person })}</DialogTitle>
                        <DialogContent>
                            <Stack spacing={1}>
                                {roles.map((r) => (
                                    <ChoiceCard key={r.id} selected={newRole === r.id} onSelect={() => setNewRole(r.id)}
                                                title={roleName(r.id)} description={t(`collaboration.roleDesc.${r.id}`, { defaultValue: '' })} />
                                ))}
                                {lostAreas.length > 0 && (
                                    <Alert severity="info">{t('collaboration.people.roleLess', { name: roleFor.person, areas: lostAreas.join(', ') })}</Alert>
                                )}
                            </Stack>
                        </DialogContent>
                        <DialogActions>
                            <Button onClick={() => setRoleFor(null)}>{t('buttons.abort')}</Button>
                            <Button variant="contained" disabled={newRole === roleFor.role} onClick={doChangeRole}>{t('buttons.save')}</Button>
                        </DialogActions>
                    </>
                )}
            </Dialog>

            <ApproveRequestDialog
                request={approving}
                onClose={(result) => {
                    if (result === 'approved') setToast(t('collaboration.request.approved', { name: approving?.person }));
                    if (result === 'rejected') setToast(t('collaboration.request.rejected'));
                    setApproving(null);
                    if (result) {
                        load();
                        refresh();
                    }
                }}
            />
            <Snackbar open={!!toast} autoHideDuration={3000} onClose={() => setToast('')} message={toast} />
        </Stack>
    );
}

function Invites({ roles, startNew }: { roles: ClubRole[]; startNew: boolean }) {
    const { t } = useTranslation();
    const roleName = useRoleName();
    const errorText = useErrorText();
    const [invites, setInvites] = useState<ClubInvite[] | null>(null);
    const [wizard, setWizard] = useState(startNew);
    const [error, setError] = useState('');
    const [toast, setToast] = useState('');

    const load = useCallback(() => listInvites().then(setInvites, (e) => setError(errorText(e))), []);
    useEffect(() => { load(); }, [load]);

    if (wizard) {
        return (
            <Paper sx={{ p: 3 }}>
                <InviteWizard roles={roles} onDone={() => { setWizard(false); load(); }} />
            </Paper>
        );
    }

    const state = (i: ClubInvite) => i.usedAt ? 'Used' : new Date(i.expiresAt) < new Date() ? 'Expired' : 'Open';

    return (
        <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography variant="h6" sx={{ flex: 1 }}>{t('collaboration.invite.open')}</Typography>
                <Button variant="contained" onClick={() => setWizard(true)}>{t('collaboration.invite.new')}</Button>
            </Box>
            {error && <Alert severity="error">{error}</Alert>}
            {invites === null ? <CircularProgress size={24} /> : invites.length === 0 ? (
                <Typography color="text.secondary">{t('collaboration.invite.none')}</Typography>
            ) : (
                <TableContainer>
                    <Table size="small">
                        <TableHead>
                            <TableRow>
                                <TableCell>{t('collaboration.invite.name')}</TableCell>
                                <TableCell>{t('collaboration.invite.role')}</TableCell>
                                <TableCell>{t('collaboration.invite.validUntil')}</TableCell>
                                <TableCell>{t('collaboration.invite.status')}</TableCell>
                                <TableCell />
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {invites.map((i) => (
                                <TableRow key={i.id}>
                                    <TableCell>{i.person}</TableCell>
                                    <TableCell>{roleName(i.role)}</TableCell>
                                    <TableCell sx={{ fontVariantNumeric: 'tabular-nums' }}>{new Date(i.expiresAt).toLocaleDateString()}</TableCell>
                                    <TableCell>{t(`collaboration.invite.status${state(i)}`)}</TableCell>
                                    <TableCell align="right">
                                        {state(i) === 'Open' && (
                                            <Button size="small" onClick={async () => {
                                                try {
                                                    await withdrawInvite(i.id);
                                                    setToast(t('collaboration.invite.withdrawn'));
                                                    load();
                                                } catch (e) {
                                                    setError(errorText(e));
                                                }
                                            }}>{t('collaboration.invite.withdraw')}</Button>
                                        )}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}
            <Snackbar open={!!toast} autoHideDuration={3000} onClose={() => setToast('')} message={toast} />
        </Paper>
    );
}

/** Settings → Zusammenarbeit (UX §3). */
export default function CollaborationSettings() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const { status } = useCollaboration();
    const [params, setParams] = useSearchParams();
    const [roles, setRoles] = useState<ClubRole[]>([]);

    const tab = (params.get('tab') as TabId) || 'overview';
    const setTab = (v: TabId) => setParams({ tab: v }, { replace: true });

    const loadRoles = useCallback(() => { listRoles().then(setRoles, () => {}); }, []);
    useEffect(loadRoles, [loadRoles, status?.mode]);

    let body: React.ReactNode;
    if (!status) {
        body = <CircularProgress size={24} />;
    } else if (status.mode === 'local') {
        body = <LocalMode />;
    } else if (status.locked) {
        body = <Alert severity="warning">{t('collaboration.errors.LOCKED')}</Alert>;
    } else if (status.mode === 'joining') {
        body = (
            <Alert severity="info" action={<Button onClick={() => navigate('/collaboration/join')}>{t('collaboration.popover.view')}</Button>}>
                {t('collaboration.status.joining')}
            </Alert>
        );
    } else if (status.mode === 'revoked') {
        body = <Alert severity="error">{t('collaboration.status.revoked')}</Alert>;
    } else {
        const tabs: TabId[] = status.isAdmin ? ['overview', 'people', 'invites', 'roles', 'emergency'] : ['overview', 'people', 'roles', 'emergency'];
        const current = tabs.includes(tab) ? tab : 'overview';
        body = (
            <Stack spacing={2}>
                <Tabs value={current} onChange={(_, v) => setTab(v)} variant="scrollable" allowScrollButtonsMobile>
                    {tabs.map((id) => (
                        <Tab key={id} value={id} label={
                            id === 'people' && status.pendingRequests > 0
                                ? <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>{t(`collaboration.settings.tabs.${id}`)}<Chip size="small" color="warning" label={status.pendingRequests} /></Box>
                                : t(`collaboration.settings.tabs.${id}`)
                        } />
                    ))}
                </Tabs>
                {current === 'overview' && <Overview roles={roles} />}
                {current === 'people' && <People roles={roles} onChanged={loadRoles} />}
                {current === 'invites' && <Invites roles={roles} startNew={params.get('new') === '1'} />}
                {current === 'roles' && (
                    <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
                        <Typography variant="h6">{t('collaboration.settings.rolesTitle')}</Typography>
                        <RoleTable roles={roles} highlight={status.role} />
                        <Typography variant="body2" color="text.secondary">{t('collaboration.settings.rolesFixed')}</Typography>
                    </Paper>
                )}
                {current === 'emergency' && (
                    <Paper sx={{ p: 3, display: 'flex', flexDirection: 'column', gap: 2 }}>
                        <Typography variant="h6">{t('collaboration.settings.emergencyTitle')}</Typography>
                        <Typography>{t('collaboration.settings.emergencyText')}</Typography>
                        <Box><Button variant="outlined" onClick={() => navigate('/collaboration/recover')}>{t('collaboration.setup.recover')}</Button></Box>
                    </Paper>
                )}
            </Stack>
        );
    }

    return (
        <Box sx={{ p: { xs: 2, sm: 4 } }}>
            <PageHeader title={t('collaboration.title')} icon={<GroupsIcon fontSize="small" />} />
            {body}
        </Box>
    );
}
