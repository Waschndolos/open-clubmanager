import { useCallback, useEffect, useState } from 'react';
import {
    Alert,
    Autocomplete,
    Box,
    Button,
    Chip,
    Grid,
    IconButton,
    List,
    ListItem,
    ListItemText,
    MenuItem,
    TextField,
    Typography,
} from '@mui/material';
import { Delete, Star } from '@mui/icons-material';
import { useTranslation } from 'react-i18next';
import { Link as RouterLink } from 'react-router-dom';
import { usePermission } from '../../context/CollaborationContext';
import { useAuth } from '../../context/AuthContext';
import { useNotification } from '../../context/NotificationContext';
import { fetchMembers } from '../../api/members';
import { Member } from '../../api/types';
import {
    createGuardian,
    deleteGuardian,
    fetchGuardians,
    fetchHouseholds,
    Guardian,
    Household,
    isMinor,
    memberName,
} from '../../api/households';

/** The household of a member, or null when it has none (undefined while loading). */
export function useMemberHousehold(memberId: string): Household | null | undefined {
    const core = usePermission('membersCore');
    const [household, setHousehold] = useState<Household | null | undefined>(undefined);
    useEffect(() => {
        if (!memberId || core === 'none') {
            setHousehold(null);
            return;
        }
        fetchHouseholds()
            .then((list) => setHousehold(list.find((h) => h.members.some((m) => m.id === memberId)) ?? null))
            .catch(() => setHousehold(null));
    }, [memberId, core]);
    return household;
}

/** Warns when the main payer of a household is about to leave the club. */
export function PayerExitWarning({ memberId, exitDate, household }: {
    memberId: string;
    exitDate?: string | Date;
    household: Household | null | undefined;
}) {
    const { t } = useTranslation();
    if (!exitDate || !household || household.payer?.id !== memberId) return null;
    const remaining = household.members.filter((m) => m.id !== memberId && m.active);
    if (remaining.length === 0) return null;
    return (
        <Alert severity="warning" sx={{ mt: 2 }}>
            {t('households.payerLeaving', { household: household.name, names: remaining.map(memberName).join(', ') })}
        </Alert>
    );
}

type Props = {
    member: Member;
    household: Household | null | undefined;
    /** Opens another member, for the links to family members. */
    onOpenMember?: (id: string) => void;
};

export function FamilySection({ member, household, onOpenMember }: Props) {
    const { t } = useTranslation();
    const core = usePermission('membersCore');
    const contact = usePermission('membersContact');

    return (
        <Box display="flex" flexDirection="column" gap={2}>
            {core !== 'none' && (
                <Box>
                    <Typography variant="subtitle2" gutterBottom>{t('households.household')}</Typography>
                    {household === null && (
                        <Typography variant="body2" color="text.secondary">
                            {t('households.noHousehold')}{' '}
                            <RouterLink to="/households">{t('households.manage')}</RouterLink>
                        </Typography>
                    )}
                    {household && (
                        <>
                            <Typography variant="body2" sx={{ mb: 1 }}>
                                <RouterLink to="/households">{household.name}</RouterLink>
                                {household.payer && <> · {t('households.columns.payer')}: {memberName(household.payer)}</>}
                            </Typography>
                            <Box display="flex" flexWrap="wrap" gap={0.5}>
                                {household.members.filter((m) => m.id !== member.id).map((m) => (
                                    <Chip key={m.id} size="small" label={memberName(m)}
                                          icon={household.payer?.id === m.id ? <Star fontSize="small" /> : undefined}
                                          variant={m.active ? 'filled' : 'outlined'}
                                          clickable={!!onOpenMember} onClick={onOpenMember ? () => onOpenMember(m.id) : undefined} />
                                ))}
                            </Box>
                            {household.payerMissing && (
                                <Alert severity="warning" sx={{ mt: 1 }}>{t('households.payerMissing')}</Alert>
                            )}
                        </>
                    )}
                </Box>
            )}
            {contact !== 'none' && <Guardians member={member} canEdit={contact === 'write'} />}
        </Box>
    );
}

function Guardians({ member, canEdit }: { member: Member; canEdit: boolean }) {
    const { t } = useTranslation();
    const { appRole } = useAuth();
    const { addNotification } = useNotification();
    const writable = canEdit && appRole !== 'READONLY';
    const [guardians, setGuardians] = useState<Guardian[]>([]);
    const [adding, setAdding] = useState(false);

    const reload = useCallback(() => {
        fetchGuardians(member.id).then(setGuardians).catch(() => addNotification(t('households.guardians.loadFailed')));
    }, [member.id, addNotification, t]);
    useEffect(reload, [reload]);

    const minor = isMinor(member.birthday);

    return (
        <Box>
            <Typography variant="subtitle2" gutterBottom>{t('households.guardians.title')}</Typography>
            {minor && guardians.length === 0 && (
                <Alert severity="info" sx={{ mb: 1 }}>{t('households.guardians.minorHint')}</Alert>
            )}
            {guardians.length === 0 && !minor && (
                <Typography variant="body2" color="text.secondary">{t('households.guardians.none')}</Typography>
            )}
            <List dense disablePadding>
                {guardians.map((g) => (
                    <ListItem key={g.id} disableGutters secondaryAction={writable && (
                        <IconButton edge="end" aria-label={t('households.guardians.delete')} onClick={() => {
                            deleteGuardian(g.id).then(reload).catch(() => addNotification(t('households.saveFailed')));
                        }}><Delete /></IconButton>
                    )}>
                        <ListItemText
                            primary={`${g.member ? memberName(g.member) : g.name ?? ''}${g.relation ? ` (${g.relation})` : ''}`}
                            secondary={[g.phone, g.email].filter(Boolean).join(' · ') || (g.member ? t('households.guardians.isMember') : undefined)} />
                    </ListItem>
                ))}
            </List>
            {writable && !adding && (
                <Button size="small" onClick={() => setAdding(true)}>{t('households.guardians.add')}</Button>
            )}
            {writable && adding && (
                <GuardianForm memberId={member.id} onDone={(saved) => {
                    setAdding(false);
                    if (saved) reload();
                }} />
            )}
        </Box>
    );
}

function GuardianForm({ memberId, onDone }: { memberId: string; onDone: (saved: boolean) => void }) {
    const { t } = useTranslation();
    const { addNotification } = useNotification();
    const [kind, setKind] = useState<'member' | 'person'>('member');
    const [members, setMembers] = useState<Member[]>([]);
    const [guardianId, setGuardianId] = useState('');
    const [name, setName] = useState('');
    const [relation, setRelation] = useState('');
    const [email, setEmail] = useState('');
    const [phone, setPhone] = useState('');

    useEffect(() => {
        fetchMembers().then((list) => setMembers(list.filter((m) => m.id !== memberId))).catch(() => undefined);
    }, [memberId]);

    const valid = kind === 'member' ? !!guardianId : !!name.trim();
    const save = () => {
        const data = kind === 'member'
            ? { memberId: guardianId, relation }
            : { name, relation, email, phone };
        createGuardian(memberId, data).then(() => onDone(true)).catch(() => addNotification(t('households.saveFailed')));
    };

    return (
        <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid size={{ xs: 12, sm: 4 }}>
                <TextField select fullWidth size="small" label={t('households.guardians.kind')} value={kind}
                           onChange={(e) => setKind(e.target.value as 'member' | 'person')}>
                    <MenuItem value="member">{t('households.guardians.kindMember')}</MenuItem>
                    <MenuItem value="person">{t('households.guardians.kindPerson')}</MenuItem>
                </TextField>
            </Grid>
            <Grid size={{ xs: 12, sm: 8 }}>
                {kind === 'member' ? (
                    <Autocomplete size="small" options={members} value={members.find((m) => m.id === guardianId) ?? null}
                                  getOptionLabel={(m) => `${m.firstName} ${m.lastName} (${m.number})`}
                                  onChange={(_e, v) => setGuardianId(v?.id ?? '')}
                                  renderInput={(p) => <TextField {...p} label={t('households.guardians.member')} />} />
                ) : (
                    <TextField fullWidth size="small" required label={t('households.guardians.name')} value={name}
                               onChange={(e) => setName(e.target.value)} />
                )}
            </Grid>
            <Grid size={{ xs: 12, sm: 4 }}>
                <TextField fullWidth size="small" label={t('households.guardians.relation')} value={relation}
                           placeholder={t('households.guardians.relationPlaceholder')} onChange={(e) => setRelation(e.target.value)} />
            </Grid>
            {kind === 'person' && <>
                <Grid size={{ xs: 12, sm: 4 }}>
                    <TextField fullWidth size="small" label={t('members.table.header.phone')} value={phone}
                               onChange={(e) => setPhone(e.target.value)} />
                </Grid>
                <Grid size={{ xs: 12, sm: 4 }}>
                    <TextField fullWidth size="small" type="email" label={t('members.table.header.email')} value={email}
                               onChange={(e) => setEmail(e.target.value)} />
                </Grid>
            </>}
            <Grid size={12} display="flex" justifyContent="flex-end" gap={1}>
                <Button size="small" onClick={() => onDone(false)}>{t('buttons.abort')}</Button>
                <Button size="small" variant="contained" disabled={!valid} onClick={save}>{t('households.guardians.add')}</Button>
            </Grid>
        </Grid>
    );
}
