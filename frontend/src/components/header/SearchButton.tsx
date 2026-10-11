import { useEffect, useState } from 'react';
import { IconButton, Tooltip } from '@mui/material';
import SearchRoundedIcon from '@mui/icons-material/SearchRounded';
import { useTranslation } from 'react-i18next';
import CommandPalette from './CommandPalette';

/** Magnifier in the header; also owns the Ctrl/⌘+K shortcut. */
export default function SearchButton() {
    const { t } = useTranslation();
    const [open, setOpen] = useState(false);

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
                e.preventDefault();
                setOpen((o) => !o);
            }
        };
        window.addEventListener('keydown', onKey);
        return () => window.removeEventListener('keydown', onKey);
    }, []);

    const label = t('search.open');
    return (
        <>
            <Tooltip title={label}>
                <IconButton color="inherit" size="small" aria-label={label} aria-haspopup="dialog" onClick={() => setOpen(true)}>
                    <SearchRoundedIcon fontSize="small" />
                </IconButton>
            </Tooltip>
            {open && <CommandPalette open={open} onClose={() => setOpen(false)} />}
        </>
    );
}
