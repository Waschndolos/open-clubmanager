import {Button, CircularProgress, TextField, Typography} from "@mui/material";
import Box from "@mui/material/Box";
import {useTranslation} from "react-i18next";
import MemberTable from "./MemberTable";
import {Member} from "./types";
import {Add} from "@mui/icons-material";
import {useEffect, useState} from "react";

const mockMembers: Member[] = [
    {id: 1, firstName: "Anna", lastName: "Müller", email: "anna@example.com"},
    {id: 2, firstName: "Max", lastName: "Meier", email: "max@example.com"},
];

export default function Members() {
    const [members, setMembers] = useState<Member[]>(mockMembers);

    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState("");
    const {t} = useTranslation();
    const filtered = members.filter((member) =>
        Object.values(member ?? {}).some((value) => {
            if (Array.isArray(value)) {
                return value.some((v) =>
                    typeof v.name === "string" &&
                    v.name.toLowerCase().includes(search.trim().toLowerCase())
                );
            }
            return String(value ?? "").toLowerCase().includes(search.trim().toLowerCase());
        })
    );


    useEffect(() => {
        fetch("http://localhost:3001/api/members") // ⇐ deine API-URL
            .then((res) => {
                if (!res.ok) throw new Error("Error fetching members.");
                return res.json();
            })
            .then((data: Member[]) => setMembers(data))
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    }, []);

    if (loading) return <CircularProgress/>;
    if (error) return <Typography color="error">{error}</Typography>;

    return (
        <Box p={3}>
            <Typography variant="h4" gutterBottom>
                {t("members.title")}
            </Typography>
            <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <TextField
                    label={t("members.search")}
                    variant="outlined"
                    size="small"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                />
                <Button variant="contained" startIcon={<Add/>}>
                    {t("members.create")}
                </Button>
            </Box>
            <MemberTable members={filtered}/>
        </Box>

    )
}
