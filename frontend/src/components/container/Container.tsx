import {Box} from "@mui/material";
import * as React from "react";

type ContainerProps = {
    spacing?: number;
    children?: React.ReactNode;
    backgroundColor?: string;
}

export default function Container({spacing = 5, backgroundColor = "transparent", children}: ContainerProps) {
    return (
        <Box
            sx={{
                width: '100%',
                height: '100%',
                p: spacing,
                backgroundColor: backgroundColor,
                boxSizing: 'border-box',
            }}
        >
            {children}
        </Box>
    );
}