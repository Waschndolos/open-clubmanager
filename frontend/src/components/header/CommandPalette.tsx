import { useEffect, useMemo, useRef, useState } from 'react';
import { Box, Dialog, DialogContent, InputBase, Typography } from '@mui/material';
import SearchRoundedIcon from '@mui/icons-material/SearchRounded';
import PersonAddAltOutlined from '@mui/icons-material/PersonAddAltOutlined';
import SyncOutlined from '@mui/icons-material/SyncOutlined';
import PersonOutline from '@mui/icons-material/PersonOutline';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { fetchMembers } from '../../api/members';
import { Member } from '../../api/types';
import { useCollaboration } from '../../context/CollaborationContext';
import { useNavGroups } from '../menu/navItems';

type Entry = {
    id: string;
    group: string;
    label: string;
    hint?: string;
    icon: React.ReactElement;
    run: () => void;
};

const MAX_MEMBERS = 8;

function matches(text: string, query: string): boolean {
    return text.toLowerCase().includes(query);
}

type Props = { open: boolean; onClose: () => void };

/** Ctrl/⌘+K palette: jump to pages and members, or run a quick action. */
export default function CommandPalette({ open, onClose }: Props) {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const groups = useNavGroups();
    const { status, sync } = useCollaboration();
    const [query, setQuery] = useState('');
    const [active, setActive] = useState(0);
    const [members, setMembers] = useState<Member[]>([]);
    const listRef = useRef<HTMLDivElement>(null);

    // Members are loaded when the palette opens, so typing filters in memory.
    useEffect(() => {
        if (!open) return;
        setQuery('');
        setActive(0);
        let cancelled = false;
        fetchMembers()
            .then((m) => { if (!cancelled) setMembers(m); })
            .catch(() => { if (!cancelled) setMembers([]); });
        return () => { cancelled = true; };
    }, [open]);

    const entries = useMemo<Entry[]>(() => {
        const q = query.trim().toLowerCase();
        const go = (path: string) => () => navigate(path);
        const pages: Entry[] = groups.flatMap((g) => g.items).map((i) => ({
            id: `page:${i.link}`, group: t('search.groups.pages'), label: i.label, icon: i.icon, run: go(i.link),
        }));
        const actions: Entry[] = [
            { id: 'action:new-member', group: t('search.groups.actions'), label: t('search.actions.newMember'), icon: <PersonAddAltOutlined />, run: go('/members?new=1') },
            ...(status && status.mode !== 'local' && !status.locked
                ? [{ id: 'action:sync', group: t('search.groups.actions'), label: t('search.actions.sync'), icon: <SyncOutlined />, run: () => { void sync(); } }]
                : []),
        ];
        const memberEntries: Entry[] = q
            ? members
                .filter((m) => matches(`${m.firstName ?? ''} ${m.lastName ?? ''}`, q) || matches(`${m.lastName ?? ''} ${m.firstName ?? ''}`, q) || (m.number != null && String(m.number).startsWith(q)))
                .slice(0, MAX_MEMBERS)
                .map((m) => ({
                    id: `member:${m.id}`,
                    group: t('search.groups.members'),
                    label: `${m.firstName ?? ''} ${m.lastName ?? ''}`.trim() || m.email,
                    hint: m.number ? `#${m.number}` : undefined,
                    icon: <PersonOutline />,
                    run: go(`/members?q=${encodeURIComponent(`${m.firstName ?? ''} ${m.lastName ?? ''}`.trim())}`),
                }))
            : [];
        const filter = (list: Entry[]) => (q ? list.filter((e) => matches(e.label, q)) : list);
        return [...filter(actions), ...memberEntries, ...filter(pages)];
    }, [query, members, groups, status, sync, navigate, t]);

    useEffect(() => { setActive(0); }, [query]);

    useEffect(() => {
        listRef.current?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' });
    }, [active]);

    const choose = (entry: Entry | undefined) => {
        if (!entry) return;
        onClose();
        entry.run();
    };

    const onKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'ArrowDown') {
            e.preventDefault();
            setActive((a) => (entries.length ? (a + 1) % entries.length : 0));
        } else if (e.key === 'ArrowUp') {
            e.preventDefault();
            setActive((a) => (entries.length ? (a - 1 + entries.length) % entries.length : 0));
        } else if (e.key === 'Enter') {
            e.preventDefault();
            choose(entries[active]);
        }
    };

    return (
        <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm" slotProps={{ paper: { sx: { position: 'absolute', top: { xs: 24, sm: 80 }, m: 0, width: 'calc(100% - 32px)' } } }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, py: 1, borderBottom: 1, borderColor: 'divider' }}>
                <SearchRoundedIcon color="action" />
                <InputBase
                    autoFocus
                    fullWidth
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    onKeyDown={onKeyDown}
                    placeholder={t('search.placeholder')}
                    role="combobox"
                    aria-expanded
                    aria-controls="command-palette-list"
                    aria-autocomplete="list"
                    aria-activedescendant={entries[active] ? `cp-${entries[active].id}` : undefined}
                    inputProps={{ 'aria-label': t('search.label') }}
                />
            </Box>
            <DialogContent sx={{ p: 0, maxHeight: 400 }}>
                <Box id="command-palette-list" role="listbox" aria-label={t('search.label')} ref={listRef} sx={{ py: 0.5 }}>
                    {entries.length === 0 && (
                        <Typography role="status" color="text.secondary" sx={{ p: 2 }}>{t('search.empty')}</Typography>
                    )}
                    {entries.map((entry, i) => (
                        <Box key={entry.id}>
                            {(i === 0 || entries[i - 1].group !== entry.group) && (
                                <Typography variant="overline" color="text.secondary" sx={{ px: 2, display: 'block' }} aria-hidden>
                                    {entry.group}
                                </Typography>
                            )}
                            <Box
                                id={`cp-${entry.id}`}
                                role="option"
                                aria-selected={i === active}
                                onMouseMove={() => setActive(i)}
                                onClick={() => choose(entry)}
                                sx={{
                                    display: 'flex', alignItems: 'center', gap: 1.5, px: 2, py: 1, cursor: 'pointer',
                                    bgcolor: i === active ? 'action.selected' : 'transparent',
                                }}
                            >
                                {entry.icon}
                                <Typography sx={{ flexGrow: 1 }} noWrap>{entry.label}</Typography>
                                {entry.hint && <Typography variant="body2" color="text.secondary">{entry.hint}</Typography>}
                            </Box>
                        </Box>
                    ))}
                </Box>
            </DialogContent>
        </Dialog>
    );
}
