import React from 'react';
import { Box, Paper, Step, StepLabel, Stepper, Typography } from '@mui/material';

/** Page frame shared by the collaboration wizards. */
export default function WizardFrame(props: { title: string; steps?: string[]; active?: number; children: React.ReactNode }) {
    const { title, steps, active = 0, children } = props;
    return (
        <Box sx={{ p: { xs: 2, sm: 4 }, display: 'flex', justifyContent: 'center' }}>
            <Paper sx={{ p: { xs: 2, sm: 4 }, width: '100%', maxWidth: 760, display: 'flex', flexDirection: 'column', gap: 3 }}>
                <Typography variant="h5" component="h1">{title}</Typography>
                {steps && (
                    <Stepper activeStep={active} alternativeLabel sx={{ overflowX: 'auto', pb: 1 }}>
                        {steps.map((s) => (
                            <Step key={s}><StepLabel>{s}</StepLabel></Step>
                        ))}
                    </Stepper>
                )}
                {children}
            </Paper>
        </Box>
    );
}
