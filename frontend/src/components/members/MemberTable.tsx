import {Delete, Edit } from "@mui/icons-material";
import {IconButton, Paper, Table, TableBody, TableCell, TableHead, TableRow} from "@mui/material";
import {Member} from "./types";
import {useTranslation} from "react-i18next";

type MemberTableProps = {
    members: Member[];
};

export default function MemberTable({members}: MemberTableProps) {
    const { t } = useTranslation();
    const visibleKeys = Object.keys(members[0]) as (keyof Member)[];
    return(
        <Paper>
            <Table>
                <TableHead>
                    <TableRow>
                        {visibleKeys.map((key) => (
                            <TableCell key={key}>{t("members.table.header." + key)}</TableCell>
                        ))}
                        <TableCell>{t("members.table.header.actions")}</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {members.map((member,index) => (
                        <TableRow key={index}>
                            {visibleKeys.map((key)=> (
                                <TableCell key={key as string}>
                                    {formatValue(member[key])}
                                </TableCell>
                            ))}
                            <TableCell>
                                <IconButton><Edit /></IconButton>
                                <IconButton><Delete /></IconButton>
                            </TableCell>
                        </TableRow>
                    ))}

                </TableBody>
            </Table>
        </Paper>
    )
}

function formatValue(value: unknown): string {
    if (value instanceof Date) return value.toLocaleDateString();
    if (typeof value === "string" || typeof value === "number") return value.toString();
    if (value === undefined || value === null) return "-";
    return JSON.stringify(value);
}