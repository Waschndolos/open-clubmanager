import React, { useState } from "react";
import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    MenuItem,
    Select,
    Typography
} from "@mui/material";
import { Member } from "../../components/api/types";

type Props = {
    excelHeaders: string[];
    excelData: (string | number | null)[][];
    targetFields: (keyof Member)[];
    onClose: () => void;
    onImport: (members: Member[]) => void;
};

type ColumnMapping = Record<string, keyof Member>;

export default function ImportMembersDialog({
                                                excelHeaders,
                                                excelData,
                                                targetFields,
                                                onClose,
                                                onImport
                                            }: Props) {
    const [mapping, setMapping] = useState<ColumnMapping>({} as ColumnMapping);

    const handleChange = (excelKey: string, value: string) => {
        setMapping((prev) => ({
            ...prev,
            [excelKey]: value as keyof Member
        }));
    };

    const handleSubmit = () => {
        const members: Member[] = excelData.map((row) => {
            const obj: Partial<Member> = {};
            excelHeaders.forEach((header, index) => {
                const mappedKey = mapping[header];
                if (mappedKey) {
                    obj[mappedKey] = row[index] as any;
                }
            });
            return obj as Member;
        });

        onImport(members);
    };

    const isValidMapping = Object.values(mapping).length > 0;

    const firstPreview = (() => {
        if (!excelData.length) return null;
        const preview: Partial<Member> = {};
        excelHeaders.forEach((header, index) => {
            const mappedKey = mapping[header];
            if (mappedKey) {
                preview[mappedKey] = excelData[0][index] as any;
            }
        });
        return preview;
    })();

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>Excel-Spalten zuordnen</DialogTitle>
            <DialogContent>
                <Typography variant="body2" gutterBottom>
                    Ordne die Spalten deiner Excel-Datei den Feldern der Mitgliederliste zu.
                </Typography>

                {excelHeaders.map((header) => (
                    <Box key={header} display="flex" alignItems="center" mb={2}>
                        <Typography sx={{ width: 200 }}>{header}</Typography>
                        <Select
                            fullWidth
                            value={mapping[header] || ""}
                            onChange={(e) => handleChange(header, e.target.value)}
                        >
                            <MenuItem value="">-- Ignorieren --</MenuItem>
                            {targetFields.map((field) => (
                                <MenuItem key={field as string} value={field}>
                                    {field}
                                </MenuItem>
                            ))}
                        </Select>
                    </Box>
                ))}

                {firstPreview && (
                    <Box mt={3}>
                        <Typography variant="subtitle2">Vorschau des ersten Datensatzes:</Typography>
                        <Box component="pre" sx={{ background: "#f5f5f5", p: 2, borderRadius: 1, fontSize: 12 }}>
                            {JSON.stringify(firstPreview, null, 2)}
                        </Box>
                    </Box>
                )}
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>Abbrechen</Button>
                <Button onClick={handleSubmit} variant="contained" disabled={!isValidMapping}>
                    Importieren
                </Button>
            </DialogActions>
        </Dialog>
    );
}
