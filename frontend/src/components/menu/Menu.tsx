import {Box, List, Typography} from "@mui/material";
import MenuItem from "./MenuItem";
import packageJson from '../../../package.json'


export default function Menu() {
    return (
        <Box sx={{
            display: 'flex',
            flexDirection: 'column',
            height: '100%',
            width: '100%',
            padding: 4,
            boxShadow: 2,
        }}>
            <List sx={{ flexGrow: 1 }}>
                <MenuItem label="Home" />
                <MenuItem label="Members" />
                <MenuItem label="Finance" />
            </List>

            <Box sx={{ marginTop: 'auto' }}>
                <Typography variant="body2" color="text.secondary" align="center">
                    Version {packageJson.version}
                </Typography>
            </Box>
        </Box>
    )
}