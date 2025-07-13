import React from 'react';
import {IconButton, Tooltip} from '@mui/material';
import LogoutIcon from '@mui/icons-material/Logout';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '../../context/AuthContext';
import {useTranslation} from "react-i18next";

const LogoutButton: React.FC = () => {
    const {t} = useTranslation();
    const navigate = useNavigate();
    const {setAccessToken} = useAuth();

    const handleLogout = () => {

        setAccessToken(null);
        localStorage.removeItem('refreshToken');

        navigate('/login', {replace: true});
    };

    return (
        <Tooltip title={t('logout.tooltip')}>
            <IconButton onClick={handleLogout} color="inherit">
                <LogoutIcon/>
            </IconButton>
        </Tooltip>
    );
};

export default LogoutButton;
