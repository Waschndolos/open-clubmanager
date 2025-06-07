import {Box, Grid} from '@mui/material';
import Menu from "../components/menu/Menu";
import Content from "../components/content/Content";
import Header from "../components/header/Header";


export default function MainLayout() {
    return (
        <>
            <Header/>
            <Box sx={{width: '100vw', height: '100vh', flexGrow: 1, padding: "3rem", overflow: 'hidden'}}>
                <Grid container spacing={2}>
                    <Grid size={3}>
                        <Menu></Menu>
                    </Grid>
                    <Grid size={9}>
                        <Content></Content>
                    </Grid>
                </Grid>
            </Box>
        </>
    );
}
