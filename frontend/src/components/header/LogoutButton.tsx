import React from 'react';
import {IconButton, Tooltip} from '@mui/material';
import LogoutIcon from '@mui/icons-material/Logout';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '../../context/AuthContext';
import {useTranslation} from "react-i18next";
import {useTheme} from "@mui/material/styles";

const LogoutButton: React.FC = () => {
    const {t} = useTranslation();
    const theme = useTheme();
    const navigate = useNavigate();
    const {setAccessToken} = useAuth();

    const handleLogout = () => {

        setAccessToken(null);
        localStorage.removeItem('refreshToken');

        navigate('/login', {replace: true});
    };

    return (
        <Tooltip title={t('logout.tooltip')}>
            <IconButton
                onClick={handleLogout}
                color="inherit"
                sx={{
                    transition: 'all 0.3s ease',
                    '&:hover': {
                        backgroundColor: theme.palette.mode === 'dark'
                            ? 'rgba(0, 255, 194, 0.1)'
                            : 'rgba(0, 200, 154, 0.1)',
                        transform: 'scale(1.05)',
                    }
                }}
            >
                <LogoutIcon/>
            </IconButton>
        </Tooltip>
    );
};

export default LogoutButton;
