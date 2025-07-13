import React from 'react';
import { IconButton, Tooltip } from '@mui/material';
import LogoutIcon from '@mui/icons-material/Logout';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

const LogoutButton: React.FC = () => {
    const navigate = useNavigate();
    const { setAccessToken } = useAuth();

    const handleLogout = () => {

        setAccessToken(null);
        localStorage.removeItem('refreshToken');

        navigate('/login', { replace: true });
    };

    return (
        <Tooltip title="Logout">
            <IconButton onClick={handleLogout} color="inherit">
                <LogoutIcon />
            </IconButton>
        </Tooltip>
    );
};

export default LogoutButton;
