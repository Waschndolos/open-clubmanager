import {Box, Grid} from '@mui/material';
import AppMenu from "../components/menu/AppMenu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";


export default function MainLayout() {
    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100vh', width: '100vw' }}>
            <Header />
            <Box sx={{ display: 'flex', flexGrow: 1, overflow: 'hidden' }}>
                <Box sx={{ width: '20%', borderRight: '1px solid #ddd' }}>
                    <AppMenu />
                </Box>
                <Box sx={{ flexGrow: 1, overflow: 'auto' }}>
                    <Content />
                </Box>
            </Box>
        </Box>
    );
}
