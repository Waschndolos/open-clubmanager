import {Button, TextField, Typography } from "@mui/material";
import Box from "@mui/material/Box";
import {useTranslation} from "react-i18next";
import MemberTable from "./MemberTable";
import {Member} from "./types";
import { Add } from "@mui/icons-material";
import {useState} from "react";

const mockMembers: Member[] = [
    { id: 1, firstName: "Anna", lastName: "Müller", email: "anna@example.com"},
    { id: 2, firstName: "Max", lastName: "Meier", email: "max@example.com" },
];

export default function Members() {
    const [members] = useState<Member[]>(mockMembers);
    const [search, setSearch] = useState("");
    const { t } = useTranslation();
    const filtered = members.filter(m =>
        m.firstName?.toLowerCase().includes(search.toLowerCase())
    );
    return (
        <Box p={3}>
            <Typography variant="h4" gutterBottom>
                {t("members.title")}
            </Typography>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <TextField
                    label= {t("members.search")}
                    variant="outlined"
                    size="small"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
                <Button variant="contained" startIcon={<Add />}>
                    Mitglied hinzufügen
                </Button>
            </Box>
            <MemberTable members={filtered}/>
        </Box>

            )
}
