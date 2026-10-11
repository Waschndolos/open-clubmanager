import {Box} from "@mui/material";
import { Outlet } from "react-router";
import CollaborationBanner from "../collaboration/CollaborationBanner";

export default function Content() {
    return (
        <Box
            component="main"
            sx={{
                display: 'flex',
                flexDirection: 'column',
                px: { xs: 2, sm: 3, lg: 5 },
                pt: { xs: 2.5, sm: 4 },
                pb: { xs: 4, sm: 6 },
                maxWidth: 1600,
                mx: 'auto',
                width: '100%',
                boxSizing: 'border-box',
            }}
        >
            <CollaborationBanner />
            <Outlet></Outlet>
        </Box>
    )
}
