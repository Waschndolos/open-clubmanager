import {Box} from "@mui/material";
import { Outlet } from "react-router";

export default function Content() {
    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            padding: 4,
            boxShadow: (theme) => theme.custom.boxShadow,
            // backgroundColor: (theme) => theme.palette.mode === 'dark' ? '#23243a' : '#fff',
            borderRadius: 0,
            transition: 'background 0.4s',
        }}>
            <Outlet></Outlet>
        </Box>
    )
}