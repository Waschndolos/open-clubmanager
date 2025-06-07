import {Delete, Edit } from "@mui/icons-material";
import {IconButton, Paper, Table, TableBody, TableCell, TableHead, TableRow} from "@mui/material";
import {Member} from "./types";
import {useTranslation} from "react-i18next";

type MemberTableProps = {
    members: Member[];
};

export default function MemberTable({members}: MemberTableProps) {
    const { t } = useTranslation();
    return(
        <Paper>
            <Table>
                <TableHead>
                    <TableRow>
                        <TableCell>{t("members.table.header.firstName")}</TableCell>
                        <TableCell>{t("members.table.header.lastName")}</TableCell>
                        <TableCell>{t("members.table.header.email")}</TableCell>
                    </TableRow>
                </TableHead>
                <TableBody>
                    {members.map((member) => (
                        <TableRow key={member.id}>
                            <TableCell>{member.firstName}</TableCell>
                            <TableCell>{member.email}</TableCell>
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