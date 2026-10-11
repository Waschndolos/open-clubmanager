import React from 'react';
import { IconButton, Tooltip } from '@mui/material';
import DarkModeOutlinedIcon from '@mui/icons-material/DarkModeOutlined';
import LightModeOutlinedIcon from '@mui/icons-material/LightModeOutlined';
import { useTranslation } from 'react-i18next';
import { useThemeContext } from '../../theme/ThemeContext';

/** Switches between the light and dark theme. */
const ThemeToggle: React.FC = () => {
  const { t } = useTranslation();
  const { mode, toggleTheme } = useThemeContext();
  const label = mode === 'dark' ? t('theme.toLight') : t('theme.toDark');

  return (
    <Tooltip title={label}>
      <IconButton onClick={toggleTheme} color="inherit" size="small" aria-label={label}>
        {mode === 'dark' ? <LightModeOutlinedIcon fontSize="small" /> : <DarkModeOutlinedIcon fontSize="small" />}
      </IconButton>
    </Tooltip>
  );
};

export default ThemeToggle;
