import { useState } from "react";
import {Member} from "../../components/api/types";
import * as XLSX from 'xlsx';
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

type Props = {
    onClose: () => void;
    onImport: (members: Member[]) => void;
};

export default function ImportMembersWizard({ onClose, onImport }: Props) {
    const [step, setStep] = useState(0);
    const [file, setFile] = useState<File | null>(null);
    const [headers, setHeaders] = useState<string[]>([]);
    const [rows, setRows] = useState<(string | number | null)[][]>([]);
    const [mapping, setMapping] = useState<Record<string, keyof Member>>({});

    const handleFileUpload = (f: File) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const data = new Uint8Array(e.target?.result as ArrayBuffer);
            const workbook = XLSX.read(data, { type: "array" });
            const sheet = workbook.Sheets[workbook.SheetNames[0]];
            const json = XLSX.utils.sheet_to_json(sheet, { header: 1 }) as any[][];
            setHeaders(json[0]);
            setRows(json.slice(1));
            setStep(1); // weiter zu Mapping
        };
        reader.readAsArrayBuffer(f);
    };

    const mappedPreview = rows.slice(0, 5).map((row) => {
        const entry: Partial<Member> = {};
        headers.forEach((header, i) => {
            const key = mapping[header];
            if (key) entry[key] = row[i] as any;
        });
        return entry;
    });

    const isMappingValid = Object.values(mapping).length > 0;

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>Mitglieder importieren</DialogTitle>
            <DialogContent>
                {step === 0 && (
                    <Box>
                        <Typography>Excel-Datei auswählen</Typography>
                        <input
                            type="file"
                            accept=".xlsx"
                            onChange={(e) => e.target.files?.[0] && handleFileUpload(e.target.files[0])}
                        />
                    </Box>
                )}

                {step === 1 && (
                    <Box>
                        <Typography>Spalten zuordnen</Typography>
                        {headers.map((h) => (
                            <Box key={h} display="flex" gap={2} mt={1}>
                                <Typography sx={{ width: 150 }}>{h}</Typography>
                                <Select
                                    fullWidth
                                    value={mapping[h] || ""}
                                    onChange={(e) =>
                                        setMapping((prev) => ({ ...prev, [h]: e.target.value as keyof Member }))
                                    }
                                >
                                    <MenuItem value="">-- Ignorieren --</MenuItem>
                                    {Object.keys({} as Member).map((k) => (
                                        <MenuItem key={k} value={k}>
                                            {k}
                                        </MenuItem>
                                    ))}
                                </Select>
                            </Box>
                        ))}
                    </Box>
                )}

                {step === 2 && (
                    <Box>
                        <Typography>Vorschau</Typography>
                        <pre>{JSON.stringify(mappedPreview, null, 2)}</pre>
                    </Box>
                )}
            </DialogContent>

            <DialogActions>
                {step > 0 && <Button onClick={() => setStep(step - 1)}>Zurück</Button>}
                {step < 2 && (
                    <Button
                        onClick={() => setStep(step + 1)}
                        disabled={step === 1 && !isMappingValid}
                        variant="contained"
                    >
                        Weiter
                    </Button>
                )}
                {step === 2 && (
                    <Button
                        variant="contained"
                        onClick={() => {
                            const imported = rows.map((row) => {
                                const entry: Partial<Member> = {};
                                headers.forEach((h, i) => {
                                    const k = mapping[h];
                                    if (k) entry[k] = row[i] as any;
                                });
                                return entry as Member;
                            });
                            onImport(imported);
                            onClose();
                        }}
                    >
                        Importieren
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
}
