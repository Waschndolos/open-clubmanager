import {Button, CircularProgress, IconButton, InputAdornment, TextField, Typography} from "@mui/material";
import Box from "@mui/material/Box";
import {useTranslation} from "react-i18next";
import MemberTable from "./MemberTable";
import {Member, canEditData} from "../../api/types";
import {useAuth} from "../../context/AuthContext";
import {usePermission} from "../../context/CollaborationContext";
import {Add, Clear, ManageAccounts} from "@mui/icons-material";
import {useCallback, useEffect, useState} from "react";
import {useSearchParams} from "react-router-dom";
import {EditMemberDialog} from "./EditMemberDialog";
import {createMember, deleteMembers, fetchMembers} from "../../api/members";
import {useNotification} from "../../context/NotificationContext";
import PageHeader from "../../components/common/PageHeader";

export function createEmptyMember(number = 1): Member {
    return {
        id: "",
        email: "",
        number,
        firstName: "",
        lastName: "",
        birthday: undefined,
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
    const [members, setMembers] = useState<Member[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const [search, setSearch] = useState("");
    const [newMemberDialogOpen, setNewMemberDialogOpen] = useState(false);
    const { addNotification } = useNotification();
    const [params, setParams] = useSearchParams();
    const [openMemberId, setOpenMemberId] = useState<string | null>(null);
    const {appRole} = useAuth();
    const canEdit = canEditData(appRole) && usePermission('membersCore') === 'write';

    // Deep links: ?new=1 opens the dialog, ?q= prefills the search (command
    // palette), ?edit=<id> opens a member (household page).
    useEffect(() => {
        const q = params.get("q");
        const wantsNew = params.get("new") === "1";
        const edit = params.get("edit");
        if (q !== null) setSearch(q);
        if (wantsNew && canEdit) setNewMemberDialogOpen(true);
        if (edit) setOpenMemberId(edit);
        if (q !== null || wantsNew || edit) setParams({}, {replace: true});
    }, [params, setParams, canEdit]);



    const clearOpenMember = useCallback(() => setOpenMemberId(null), []);

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
        // Match by id: members without an e-mail would otherwise all match.
        setMembers(prev => prev.some(m => m.id === updated.id)
            ? prev.map(m => m.id === updated.id ? updated : m)
            : [...prev, updated]);
    };

    const handleMemberDeleted = (deleted: Member[]) => {
        for(const m of deleted) {
            setMembers((prevMembers) =>
                prevMembers.filter((member) => member.id !== m.id)
            );
        }
    }

    useEffect(() => {
        fetchMembers().then((members) => {
            setLoading(false);
            setMembers(members);
        }).catch((e: unknown) => {
            setLoading(false);
            setError(e instanceof Error ? e.message : String(e));
        });
    }, []);

    // Automatic member deletion, by logins that may delete members
    useEffect(() => {
        if (!canEdit) return;
        const now = new Date();

        setMembers((prev) => {
            const stillActive: Member[] = [];

            for (const member of prev) {
                const expired =
                    member.exitDate && new Date(member.exitDate) < now;

                if (expired) {
                    deleteMembers([member]).then( ()=> {
                        addNotification(
                            `${member.firstName} ${member.lastName} ${t('members.automatedDeletionNotice')}`
                        );
                    });

                } else {
                    stillActive.push(member);
                }
            }

            return stillActive;
        });
    }, [t, addNotification, members.length, canEdit]);

    // Errors propagate to the dialog, which shows them.
    const handleMemberCreated = async (member: Member) => {
        const created = await createMember(member);
        handleMemberUpdated(created);
        setNewMemberDialogOpen(false);
    };

    const nextMemberNumber = members.reduce((max, m) => Math.max(max, m.number ?? 0), 0) + 1;

    if (loading) return <CircularProgress/>;
    if (error) return <Typography color="error">{error}</Typography>;

    return (
        <Box p={3}>
            <PageHeader
                title={t("members.title")}
                icon={<ManageAccounts fontSize="small" />}
                actions={canEdit && (
                    <Button variant="contained" startIcon={<Add/>} onClick={() => setNewMemberDialogOpen(true)}>
                        {t("members.create")}
                    </Button>
                )}
            />
            <Box display="flex" justifyContent="flex-start" alignItems="center" mb={2}>
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
            </Box>
            <MemberTable members={filtered} onMemberUpdated={handleMemberUpdated} onMembersDeleted={handleMemberDeleted}
                         openMemberId={openMemberId} onOpened={clearOpenMember}/>
            {newMemberDialogOpen && (
                <EditMemberDialog
                    member={createEmptyMember(nextMemberNumber)}
                    isNew
                    onClose={() => setNewMemberDialogOpen(false)}
                    onSave={handleMemberCreated}
                />
            )}
        </Box>

    )
}
