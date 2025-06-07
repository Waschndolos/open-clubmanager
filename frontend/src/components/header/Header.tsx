import {AppBar, Box, IconButton, Menu, MenuItem, Toolbar, Typography} from '@mui/material'
import {DarkMode, LanguageOutlined, LanguageTwoTone, LightMode} from '@mui/icons-material'
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
        i18nInstance.changeLanguage(lng)
        setAnchorEl(null)
    }

    return (
        <AppBar position="static" color="primary">
            <Toolbar>
                <Typography variant="h6" sx={{flexGrow: 1}}>
                    Open ClubManager
                </Typography>

                <Box display="flex" alignItems="center" gap={1}>
                    <IconButton onClick={handleLanguageClick} color={"secondary"}>
                        <LanguageTwoTone></LanguageTwoTone>
                    </IconButton>
                    <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
                        <MenuItem onClick={() => handleLanguageChange('de')}>🇩🇪 Deutsch</MenuItem>
                        <MenuItem onClick={() => handleLanguageChange('en')}>🇬🇧 English</MenuItem>
                    </Menu>

                    <IconButton onClick={toggleTheme} color={"secondary"}>
                        {mode === 'light' ? <DarkMode/> : <LightMode/>}
                    </IconButton>
                </Box>
            </Toolbar>
        </AppBar>
    )
}
