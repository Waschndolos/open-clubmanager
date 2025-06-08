import {Button, CircularProgress, IconButton, InputAdornment, TextField, Typography} from "@mui/material";
import Box from "@mui/material/Box";
import {useTranslation} from "react-i18next";
import MemberTable from "./MemberTable";
import {Member} from "../../components/api/types";
import {Add, Clear} from "@mui/icons-material";
import {useEffect, useState} from "react";
import {EditMemberDialog} from "./EditMemberDialog";

const mockMembers: Member[] = [
    {id: 1, firstName: "Anna", lastName: "Müller", email: "anna@example.com"},
    {id: 2, firstName: "Max", lastName: "Meier", email: "max@example.com"},
];

export function createEmptyMember(): Member {
    return {
        id: 0,
        email: "",
        number: 0,
        firstName: "",
        lastName: "",
        birthday: "",
        phone: "",
        phoneMobile: "",
        comment: "",
        entryDate: undefined,
        exitDate: undefined,
        street: "",
        postalCode: "",
        city: "",
        state: "",
        accountHolder: "",
        iban: "",
        bic: "",
        bankName: "",
        sepaMandateDate: undefined,
        roles: [],
        groups: [],
        sections: []
    };
}


export default function Members() {
    const {t} = useTranslation();
    const [members, setMembers] = useState<Member[]>(mockMembers);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState("");
    const [newMemberDialogOpen, setNewMemberDialogOpen] = useState(false);

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

    const handleMemberUpdated = (updated: Member) => {
        setMembers(prev => prev.map(m => m.id === updated.id ? updated : m));
    };

    const fetchMembers = () => {
        fetch("http://localhost:3001/api/members")
            .then((res) => {
                if (!res.ok) throw new Error("Error fetching members.");
                return res.json();
            })
            .then((data: Member[]) => setMembers(data))
            .catch((err) => setError(err.message))
            .finally(() => setLoading(false));
    };

    useEffect(() => {
        fetchMembers();
    }, []);

    const handleMemberCreated = async (member: Member) => {
        try {
            const res = await fetch("http://localhost:3001/api/members", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(member),
            });

            if (!res.ok) throw new Error("Error creating member.");
            fetchMembers();
        } catch (err) {
            console.error(err);
        } finally {
            setNewMemberDialogOpen(false);
        }
    };

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
                    slotProps={{
                        input: {
                            endAdornment: search && (
                                <InputAdornment position="end">
                                    <IconButton
                                        onClick={() => setSearch("")}
                                        aria-label="clear"
                                        size="small"
                                    >
                                        <Clear fontSize="small"/>
                                    </IconButton>
                                </InputAdornment>
                            ),
                        }
                    }}
                />
                <Button variant="contained" startIcon={<Add/>} onClick={() => setNewMemberDialogOpen(true)}>
                    {t("members.create")}
                </Button>
            </Box>
            <MemberTable members={filtered} onMemberUpdated={handleMemberUpdated}/>
            {newMemberDialogOpen && (
                <EditMemberDialog
                    member={createEmptyMember()}
                    isNew
                    onClose={() => setNewMemberDialogOpen(false)}
                    onSave={handleMemberCreated}
                />
            )}
        </Box>

    )
}
