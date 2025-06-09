import {AppBar, Box, IconButton, Menu, MenuItem, Toolbar, Typography} from '@mui/material'
import {DarkMode, LightMode} from '@mui/icons-material'
import {useThemeContext} from '../../theme/ThemeContext'
import {useTranslation} from "react-i18next";
import React, {useState} from "react";

export default function Header() {
    const {i18n: i18nInstance} = useTranslation()
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
    const {mode, toggleTheme} = useThemeContext()

    const handleLanguageClick = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget)
    }

    const handleLanguageChange = (lng: string) => {
        i18nInstance.changeLanguage(lng).then(() => setAnchorEl(null))
    }
    const availableLanguages = Object.keys(i18nInstance.options.resources ?? {});

    const getIcon = (lng: string) => {
        switch (lng) {
            case 'en':
                return '🇬🇧'
            case 'de':
                return '🇩🇪'
            case 'fr':
                return '🇫🇷'
        }
        return '<UNK>'
    }

    return (
        <AppBar position="static" color="primary">
            <Toolbar>
                <Typography variant="h6" sx={{flexGrow: 1}}>
                    Open ClubManager
                </Typography>

                <Box display="flex" alignItems="center" gap={1}>
                    <IconButton onClick={handleLanguageClick} color={"secondary"}>
                        <span style={{ fontSize: '1.2rem' }}>
                            {getIcon(i18nInstance.language)}
                        </span>
                    </IconButton>
                    <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
                        {availableLanguages.map((code: string) => (
                            <MenuItem key={code} onClick={() => handleLanguageChange(code)}>
                                {getIcon(code)}
                            </MenuItem>
                        ))}
                    </Menu>

                    <IconButton onClick={toggleTheme} color={"secondary"}>
                        {mode === 'light' ? <DarkMode/> : <LightMode/>}
                    </IconButton>
                </Box>
            </Toolbar>
        </AppBar>
    )
}
