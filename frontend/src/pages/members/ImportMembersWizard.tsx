import { useState } from "react";
import { Member } from "../../api/types";
import * as XLSX from "xlsx";
import {
    Alert,
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    MenuItem,
    Select,
    Typography,
} from "@mui/material";
import { useTranslation } from "react-i18next";

export type ImportMatchField = "email" | "number";

type Props = {
    onClose: () => void;
    onImport: (members: Member[], matchField: ImportMatchField) => void;
};

const memberFields: (keyof Member)[] = [
    "number",
    "firstName",
    "lastName",
    "email",
    "phone",
    "phoneMobile",
    "birthday",
    "street",
    "postalCode",
    "city",
    "state",
    "entryDate",
    "exitDate",
    "accountHolder",
    "iban",
    "bic",
    "bankName",
    "sepaMandateDate",
    "roles",
    "groups",
    "sections",
    "comment",
];

const importMatchFields: ImportMatchField[] = ["email", "number"];

export default function ImportMembersWizard({ onClose, onImport }: Props) {
    const [step, setStep] = useState(0);
    const [headers, setHeaders] = useState<string[]>([]);
    const [rows, setRows] = useState<(string | number | null)[][]>([]);
    // mapping erlaubt jetzt auch "", um "Ignorieren" abzubilden
    const [mapping, setMapping] = useState<Record<string, keyof Member | "">>({});
    const [matchField, setMatchField] = useState<ImportMatchField>("email");
    const { t } = useTranslation();

    const handleFileUpload = (f: File) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const data = new Uint8Array(e.target?.result as ArrayBuffer);
            const workbook = XLSX.read(data, { type: "array" });
            const sheet = workbook.Sheets[workbook.SheetNames[0]];
            const json = XLSX.utils.sheet_to_json(sheet, { header: 1 }) as (string | number | null)[][];
            const headers = json[0].map(String);

            const initialMapping: Record<string, keyof Member | ""> = {};
            headers.forEach((header) => {
                initialMapping[header] = guessMapping(header);
            });

            setHeaders(headers);
            setRows(json.slice(1));
            setMapping(initialMapping);
            setStep(1);
        };
        reader.readAsArrayBuffer(f);
    };


    const mappedPreview = rows.slice(0, 5).map((row) => {
        const entry: Partial<Member> = {};
        headers.forEach((header, i) => {
            const key = mapping[header];
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-expect-error
            if (key && key !== "") entry[key] = row[i] as never;
        });
        return entry;
    });

    const isMappingValid =
        Object.values(mapping).includes("email") &&
        Object.values(mapping).includes(matchField);

    const matchFieldLabel = t(`members.dialogs.import.2.keys.${matchField}`);

    const hasDuplicateMatchFieldValues = (() => {
        const matchFieldIndex = headers.findIndex((h) => mapping[h] === matchField);
        if (matchFieldIndex === -1) return false;

        const values = rows
            .map((row) => row[matchFieldIndex])
            .map((value) => {
                if (typeof value === "string") {
                    const trimmed = value.trim();
                    return trimmed ? trimmed.toLowerCase() : null;
                }

                if (typeof value === "number") {
                    return String(value);
                }

                return null;
            })
            .filter((value): value is string => value !== null);

        return new Set(values).size !== values.length;
    })();

    function normalize(s: string): string {
        return s
            .toLowerCase()
            .replace(/[^a-z0-9]/g, "")
            .trim();
    }

    function guessMapping(header: string): keyof Member | "" {
        const normalizedHeader = normalize(header);
        for (const field of memberFields) {
            if (normalizedHeader.includes(normalize(field))) {
                return field;
            }
        }
        return "";
    }


    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>
                {t(
                    step === 0
                        ? "members.dialogs.import.1.title"
                        : step === 1
                            ? "members.dialogs.import.2.title"
                            : "members.dialogs.import.3.title"
                )}
            </DialogTitle>

            <DialogContent>
                {step === 0 && (
                    <Box>
                        <Typography>{t("members.dialogs.import.1.select")}</Typography>
                        <input
                            type="file"
                            accept=".xlsx"
                            onChange={(e) =>
                                e.target.files?.[0] && handleFileUpload(e.target.files[0])
                            }
                        />
                    </Box>
                )}

                {step === 1 && (
                    <Box>
                        <Typography>{t("members.dialogs.import.2.description")}</Typography>
                        <Box display="flex" gap={2} mt={2} alignItems="center">
                            <Typography sx={{ width: 150 }}>
                                {t("members.dialogs.import.2.matchField")}
                            </Typography>
                            <Select
                                fullWidth
                                value={matchField}
                                onChange={(e) => setMatchField(e.target.value as ImportMatchField)}
                            >
                                {importMatchFields.map((field) => (
                                    <MenuItem key={field} value={field}>
                                        {t(`members.dialogs.import.2.keys.${field}`)}
                                    </MenuItem>
                                ))}
                            </Select>
                        </Box>
                        {headers.map((h) => {
                            // Alle aktuell gewählten Felder außer dem für das aktuelle Header h
                            const selectedFields = Object.entries(mapping)
                                .filter(([key]) => key !== h) // andere Header
                                .map(([, value]) => value)
                                .filter((v) => v !== "" && v !== undefined);

                            return (
                                <Box key={h} display="flex" gap={2} mt={1} alignItems="center">
                                    <Typography sx={{ width: 150 }}>{h}</Typography>
                                    <Select
                                        fullWidth
                                        value={mapping[h] || ""}
                                        onChange={(e) =>
                                            setMapping((prev) => ({
                                                ...prev,
                                                [h]: e.target.value as keyof Member | "",
                                            }))
                                        }
                                    >
                                        <MenuItem value="">{t("members.dialogs.import.2.ignore")}</MenuItem>
                                        {memberFields
                                            .filter((k) => !selectedFields.includes(k)) // nur Felder die noch nicht gewählt sind
                                            .map((k) => (
                                                <MenuItem key={k} value={k}>
                                                    {k}
                                                </MenuItem>
                                            ))}
                                    </Select>
                                </Box>
                            );
                        })}

                    </Box>
                )}

                {step === 2 && (
                    <Box>
                        {hasDuplicateMatchFieldValues && (
                            <Alert severity="error" sx={{ mb: 2 }}>
                                {t("members.dialogs.import.3.duplicateMatchField", {
                                    field: matchFieldLabel,
                                })}
                            </Alert>
                        )}
                        <Typography>{t("members.dialogs.import.3.title")}</Typography>
                        <pre style={{ maxHeight: 300, overflow: "auto" }}>
              {JSON.stringify(mappedPreview, null, 2)}
            </pre>
                    </Box>
                )}
            </DialogContent>

            <DialogActions>
                {step > 0 && (
                    <Button onClick={() => setStep(step - 1)}>
                        {t("buttons.back")}
                    </Button>
                )}
                {step < 2 && (
                    <Button
                        onClick={() => setStep(step + 1)}
                        disabled={!isMappingValid}
                        variant="contained"
                    >
                        {t("buttons.next")}
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
                                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                    // @ts-expect-error
                                    if (k && k !== "") entry[k] = row[i] as never;
                                });
                                return entry as Member;
                            });
                            onImport(imported, matchField);
                            onClose();
                        }}
                        disabled={hasDuplicateMatchFieldValues}
                    >
                        {t("buttons.import")}
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
}
