import React from 'react';
import {
    Box,
    Radio,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Typography,
} from '@mui/material';
import { useTranslation } from 'react-i18next';
import { AREAS, AreaPermission, AreaPermissions, ClubRole, CollaborationError } from '../../api/collaboration';

/** Turns an error of the collaboration API into a sentence for the user. */
export function useErrorText(): (e: unknown) => string {
    const { t, i18n } = useTranslation();
    return (e: unknown) => {
        if (e instanceof CollaborationError) {
            const key = `collaboration.errors.${e.code}`;
            if (i18n.exists(key)) return t(key);
            return t('collaboration.errors.default', { message: e.message });
        }
        return t('collaboration.errors.default', { message: e instanceof Error ? e.message : String(e) });
    };
}

/** Display name of a role id; unknown (custom) roles show their id. */
export function useRoleName(): (id?: string) => string {
    const { t, i18n } = useTranslation();
    return (id?: string) => {
        if (!id) return '';
        const key = `collaboration.roles.${id}`;
        return i18n.exists(key) ? t(key) : id;
    };
}

/** Copies text, calling back with whether it worked. */
export async function copyText(text: string): Promise<boolean> {
    try {
        await navigator.clipboard.writeText(text);
        return true;
    } catch {
        return false;
    }
}

const permColor: Record<AreaPermission, string> = {
    write: 'primary.main',
    read: 'text.primary',
    none: 'text.disabled',
};

function PermCell({ p }: { p: AreaPermission }) {
    const { t } = useTranslation();
    return (
        <TableCell sx={{ color: permColor[p], fontWeight: p === 'none' ? 400 : 600, whiteSpace: 'nowrap' }}>
            {t(`collaboration.perm.${p}`)}
        </TableCell>
    );
}

/** One row: what a role may see and change per area. */
export function PermissionRow({ permissions }: { permissions: AreaPermissions }) {
    const { t } = useTranslation();
    return (
        <TableContainer>
            <Table size="small">
                <TableHead>
                    <TableRow>
                        {AREAS.map((a) => (
                            <TableCell key={a} sx={{ color: 'text.secondary', whiteSpace: 'nowrap' }}>
                                {t(`collaboration.areas.${a}`)}
                            </TableCell>
                        ))}
                    </TableRow>
                </TableHead>
                <TableBody>
                    <TableRow>
                        {AREAS.map((a) => <PermCell key={a} p={permissions[a]} />)}
                    </TableRow>
                </TableBody>
            </Table>
        </TableContainer>
    );
}

/** All roles with their permissions; the selected role is highlighted. */
export function RoleTable({ roles, highlight }: { roles: ClubRole[]; highlight?: string }) {
    const { t } = useTranslation();
    const roleName = useRoleName();
    return (
        <TableContainer>
            <Table size="small">
                <TableHead>
                    <TableRow>
                        <TableCell />
                        {AREAS.map((a) => (
                            <TableCell key={a} sx={{ color: 'text.secondary', whiteSpace: 'nowrap' }}>
                                {t(`collaboration.areas.${a}`)}
                            </TableCell>
                        ))}
                    </TableRow>
                </TableHead>
                <TableBody>
                    {roles.map((r) => (
                        <TableRow key={r.id} selected={r.id === highlight}>
                            <TableCell sx={{ fontWeight: 600, whiteSpace: 'nowrap' }}>{roleName(r.id)}</TableCell>
                            {AREAS.map((a) => <PermCell key={a} p={r.permissions[a]} />)}
                        </TableRow>
                    ))}
                </TableBody>
            </Table>
        </TableContainer>
    );
}

/** A selectable card with a radio, a title and a description. */
export function ChoiceCard(props: {
    selected: boolean;
    onSelect: () => void;
    title: React.ReactNode;
    description?: React.ReactNode;
    children?: React.ReactNode;
}) {
    const { selected, onSelect, title, description, children } = props;
    return (
        <Box
            role="radio"
            aria-checked={selected}
            tabIndex={0}
            onClick={onSelect}
            onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    onSelect();
                }
            }}
            sx={{
                display: 'flex',
                gap: 1,
                alignItems: 'flex-start',
                p: 1.5,
                borderRadius: 2,
                border: 1,
                borderColor: selected ? 'primary.main' : 'divider',
                bgcolor: selected ? 'action.selected' : 'transparent',
                cursor: 'pointer',
                '&:focus-visible': { outline: 2, outlineColor: 'primary.main', outlineStyle: 'solid' },
            }}
        >
            <Radio checked={selected} size="small" sx={{ p: 0.25 }} tabIndex={-1} />
            <Box sx={{ minWidth: 0, flex: 1 }}>
                <Typography sx={{ fontWeight: 600, wordBreak: 'break-word' }}>{title}</Typography>
                {description && <Typography variant="body2" color="text.secondary">{description}</Typography>}
                {children}
            </Box>
        </Box>
    );
}

/** Large, grouped safety code that is easy to read aloud. */
export function SafetyCodeDisplay({ code }: { code: string }) {
    const formatted = code.length === 6 ? `${code.slice(0, 3)} ${code.slice(3)}` : code;
    return (
        <Typography
            aria-label={code.split('').join(' ')}
            sx={{
                fontFamily: '"JetBrains Mono", ui-monospace, monospace',
                fontWeight: 700,
                fontSize: '2.5rem',
                letterSpacing: '0.08em',
                textAlign: 'center',
                py: 1.5,
                borderRadius: 2,
                bgcolor: 'action.hover',
            }}
        >
            {formatted}
        </Typography>
    );
}
