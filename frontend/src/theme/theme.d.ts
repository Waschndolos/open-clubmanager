// src/theme.d.ts

// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { ThemeOptions, Theme } from '@mui/material/styles';

declare module '@mui/material/styles' {
    interface Theme {
        custom: {
            border: string;
            boxShadow: string;
        };
    }
    interface ThemeOptions {
        custom?: {
            border?: string;
            boxShadow?: string;
        };
    }
}
