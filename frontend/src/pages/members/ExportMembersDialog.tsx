import {Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle,} from "@mui/material";
import {useTranslation} from "react-i18next";
import {Member} from "../../components/api/types";
import * as XLSX from 'xlsx';

type Props = {
    members: Member[];
    onClose: () => void;
};

export function ExportMembersDialog({ members, onClose }: Props) {
    const { t } = useTranslation();

    const flattenObject = (
        obj: Record<string, unknown>,
        prefix = ''
    ): Record<string, unknown> =>
        Object.keys(obj).reduce((acc, k) => {
            const pre = prefix ? `${prefix}.${k}` : k;
            const value = obj[k as keyof typeof obj];

            if (Array.isArray(value)) {
                acc[pre] = value.map((item) =>
                    typeof item === 'object' && item !== null
                        ? (item as Record<string, unknown>)?.name ?? JSON.stringify(item)
                        : item
                ).join(', ');
            } else if (typeof value === 'object' && value !== null) {
                Object.assign(
                    acc,
                    flattenObject(value as Record<string, unknown>, pre)
                );
            } else {
                acc[pre] = value;
            }

            return acc;
        }, {} as Record<string, unknown>);

    const exportToExcel = (data: Member[], fileName: string) => {
        const flattened = data.map(member => flattenObject(member));
        const worksheet = XLSX.utils.json_to_sheet(flattened);
        const workbook = XLSX.utils.book_new();
        XLSX.utils.book_append_sheet(workbook, worksheet, 'Daten');
        XLSX.writeFile(workbook, fileName);
    };

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>
                {t("members.dialogs.export.title")}
            </DialogTitle>
            <DialogContent>
             <DialogContentText>{t("members.dialogs.export.description")}</DialogContentText>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button
                    variant="contained"
                    onClick={() => {
                        exportToExcel(members, "memberlist.xlsx");
                    }}
                >
                    {t("buttons.export")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
