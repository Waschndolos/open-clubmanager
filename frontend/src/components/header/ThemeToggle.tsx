import React from 'react';
import { IconButton, Tooltip } from '@mui/material';
import { useThemeContext } from '../../theme/ThemeContext';
import { Brightness4, Brightness7 } from '@mui/icons-material';

/**
 * ThemeToggle component
 * Shows a sun/moon icon depending on the current theme.
 * Animates the icon and provides accessible tooltip.
 * Uses the global theme context.
 */
const ThemeToggle: React.FC = () => {
  const { mode, toggleTheme } = useThemeContext();

  return (
    <Tooltip title={mode === 'dark' ? 'Change to light mode' : 'Change to dark mode'}>
      <IconButton
        onClick={toggleTheme}
        color="inherit"
        sx={{
          transition: 'color 0.3s, background 0.3s',
          borderRadius: '50%',
          background: mode === 'dark' ? 'rgba(142, 202, 230, 0.12)' : 'rgba(255, 183, 3, 0.10)',
          boxShadow: mode === 'dark' ? '0 2px 8px 0 rgba(142,202,230,0.15)' : '0 2px 8px 0 rgba(255,183,3,0.10)',
          '&:hover': {
            background: mode === 'dark' ? 'rgba(142, 202, 230, 0.22)' : 'rgba(255, 183, 3, 0.18)',
            color: mode === 'dark' ? '#ffb703' : '#8ecae6',
          },
        }}
        aria-label="Toggle theme"
      >
        <span
          style={{
            display: 'inline-flex',
            transition: 'transform 0.5s cubic-bezier(.68,-0.55,.27,1.55)',
            transform: mode === 'dark' ? 'rotate(-40deg) scale(1.15)' : 'rotate(0deg) scale(1)',
            color: mode === 'dark' ? '#8ecae6' : '#ffb703',
            filter: mode === 'dark' ? 'drop-shadow(0 0 6px #8ecae6)' : 'drop-shadow(0 0 4px #ffb703)',
          }}
        >
          {mode === 'dark' ? <Brightness7 fontSize="medium" /> : <Brightness4 fontSize="medium" />}
        </span>
      </IconButton>
    </Tooltip>
  );
};

export default ThemeToggle; 