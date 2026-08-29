import React from 'react';
import { IconButton, Tooltip } from '@mui/material';
import { useThemeContext } from '../../theme/ThemeContext';
import {Brightness4, Brightness7} from '@mui/icons-material';

/**
 * Compact theme toggle button with a subtle animated icon.
 */
const ThemeToggle: React.FC = () => {
  const { mode, toggleTheme } = useThemeContext();

  return (
    <Tooltip title={mode === 'dark' ? 'Change to light mode' : 'Change to dark mode'}>
      <IconButton
        onClick={toggleTheme}
        color="inherit"
        size="small"
        aria-label="Toggle theme"
      >
        <span
          style={{
            display: 'inline-flex',
            transition: 'transform 0.4s cubic-bezier(.68,-0.55,.27,1.55)',
            transform: mode === 'dark' ? 'rotate(-30deg) scale(1.1)' : 'rotate(0deg) scale(1)',
          }}
        >
          {mode === 'dark' ? <Brightness7 fontSize="small" /> : <Brightness4 fontSize="small" />}
        </span>
      </IconButton>
    </Tooltip>
  );
};

export default ThemeToggle;
