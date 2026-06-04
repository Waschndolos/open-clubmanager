import {Box} from "@mui/material";
import { Outlet } from "react-router";

export default function Content() {
    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            padding: { xs: 2, sm: 3, md: 4, lg: 5 },
            borderRadius: 0,
            backgroundColor: (theme) => theme.palette.background.default,
            transition: 'background 0.4s cubic-bezier(0.4, 0, 0.2, 1)',
            overflowY: 'auto',
        }}>
            <Outlet></Outlet>
        </Box>
    )
}