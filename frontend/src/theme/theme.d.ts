// src/theme.d.ts
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
