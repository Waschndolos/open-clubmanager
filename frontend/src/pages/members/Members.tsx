import {Button, CircularProgress, IconButton, InputAdornment, TextField, Typography} from "@mui/material";
import Box from "@mui/material/Box";
import {useTranslation} from "react-i18next";
import MemberTable from "./MemberTable";
import {Member} from "../../components/api/types";
import {Add, Clear} from "@mui/icons-material";
import {useEffect, useState} from "react";
import {EditMemberDialog} from "./EditMemberDialog";
import {createMember, fetchMembers} from "../../components/api/members";
import log from "eslint-plugin-react/lib/util/log";

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
        if (members.find((member) => member.id === updated.id)) {
            setMembers(prev => prev.map(m => m.id === updated.id ? updated : m));
        } else {
            setMembers([...members, updated])
        }
    };

    useEffect(() => {
        fetchMembers().then((members) => {
            setLoading(false);
            setMembers(members);
        }).catch(setError);
    }, []);

    const handleMemberCreated = async (member: Member) => {
        createMember(member).then((member) => {
            setLoading(false);
            handleMemberUpdated(member);
            setNewMemberDialogOpen(false);
        })
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
