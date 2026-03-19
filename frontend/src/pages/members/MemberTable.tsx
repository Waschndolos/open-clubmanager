import {Backdrop, Box, CircularProgress, IconButton, Tooltip, Typography} from "@mui/material";
import {useTranslation} from "react-i18next";
import React, {useEffect, useMemo, useState} from "react";
import {ClubSection, Group, Member, Role} from "../../api/types";
import {Delete, FileDownload, FileUpload} from "@mui/icons-material";
import {EditMemberDialog} from "./EditMemberDialog";
import EditIcon from '@mui/icons-material/Edit';
import {createMember, deleteMembers, updateMember} from "../../api/members";
import {fetchRoles, createRole} from "../../api/roles";
import {fetchGroups, createGroup} from "../../api/groups";
import {fetchSections, createSection} from "../../api/sections";
import {useUserPreference} from "../../hooks/useUserPreference";
import {DateRenderer, DefaultRenderer, MemberContainingNamedArtifactRenderer} from "./renderer";
import {DeletingMemberDialog} from "./DeletingMemberDialog";
import {ExportMembersDialog} from "./ExportMembersDialog";
import ImportMembersWizard from "./ImportMembersWizard";
import {useTheme} from '@mui/material/styles';
import {DataGrid, GridColDef, GridColumnVisibilityModel, GridRowSelectionModel} from '@mui/x-data-grid';
import type {GridRenderCellParams} from "@mui/x-data-grid/models/params/gridCellParams";
import {useNotification} from "../../context/NotificationContext";


type MemberTableProps = {
    members: Member[];
    onMemberUpdated: (newMember: Member) => void;
    onMembersDeleted: (deletedMember: Member[]) => void;
};

export default function MemberTable({members, onMemberUpdated, onMembersDeleted}: MemberTableProps) {
    const {t} = useTranslation();
    const { addNotification } = useNotification();
    const [editingMember, setEditingMember] = useState<Member | null>(null);
    const [exportOpened, setExportOpened] = useState<boolean>(false);
    const [deletingMembers, setDeletingMembers] = useState<Member[] | null>(null);
    const [selectedMembers, setSelectedMembers] = useState<Member[] | null>(null);
    const {getPreference, setPreference} = useUserPreference();
    const [importWizardOpen, setImportWizardOpen] = useState(false);
    const [importInProgress, setImportInProgress] = useState(false);
    const [importProgress, setImportProgress] = useState(0);
    const [columnVisibility, setColumnVisibility] = useState<GridColumnVisibilityModel>({});

    const keys = useMemo(() => {
        if (!members || members.length === 0) return [];
        return Object.keys(members[0]) as (keyof Member)[];
    }, [members]);
    const theme = useTheme();

    const columnDefs: GridColDef[] = useMemo(() => {
        if (!members || members.length === 0) return [];

        const keys = Object.keys(members[0]) as (keyof Member)[];

        const getCellRenderer = (key: keyof Member) => {
            switch (key) {
                case "entryDate":
                case "exitDate":
                case "birthday":
                    return DateRenderer;
                case "roles":
                case "groups":
                case "sections":
                    return MemberContainingNamedArtifactRenderer;
                default:
                    return DefaultRenderer;
            }
        };

        return keys.map((key) => ({
            field: key,
            headerName: t("members.table.header." + key),
            flex: 1,
            minWidth: 150,
            sortable: true,
            renderCell: (params: GridRenderCellParams) => {
                const Renderer = getCellRenderer(key);
                return <Renderer value={params.value} row={params.row} />;
            }
        }));
    }, [members, t]);


    useEffect(() => {
        if (!keys.length) return;
        getPreference("memberTableColumns").then((stored) => {
            const fallback = Object.fromEntries(keys.map((k, i) => [k, i < 10]));
            const merged = { ...fallback, ...(stored || {}) };
            setColumnVisibility(merged);
        });
    }, [getPreference, keys]);

    const handleSelectionChanged = (selectionModel: GridRowSelectionModel) => {
        const selected = members.filter(member =>
            selectionModel.ids.has(member.id) || selectionModel.ids.has(String(member.id))
        );

        setSelectedMembers(selected.length ? selected : null);
    };

    const handleColumnToggle = (visibilityModel: GridColumnVisibilityModel) => {
        setColumnVisibility(visibilityModel);
        setPreference("memberTableColumns", visibilityModel);
    };

    const handleEditMember = () => {
        if (selectedMembers?.length === 1) {
            setEditingMember(selectedMembers[0])
        }
    }

    const handleDeleteMembers = () => {
        setDeletingMembers(selectedMembers);
    }

    const handleExport = () => {
        setExportOpened(true);
    }

    const handleSaveEdit = async (updated: Member) => {
        try {
            const updatedMember = await updateMember(updated);
            onMemberUpdated(updatedMember);
            setEditingMember(null);
        } catch (err) {
            console.error('Failed to update member:', err);
            addNotification(t('members.errors.updateFailed'));
        }
    };

    async function resolveEntityIds<T extends { id: number; name: string }>(
        value: T[] | string | undefined,
        existingList: T[],
        createFn: (data: { name: string }) => Promise<T>
    ): Promise<number[]> {
        if (value === undefined || value === null) return [];

        if (typeof value === 'string') {
            const names = value.split(',').map((s) => s.trim()).filter(Boolean);
            const ids: number[] = [];
            for (const name of names) {
                const found = existingList.find((e) => e.name === name);
                if (found) {
                    ids.push(found.id);
                } else {
                    const created = await createFn({ name });
                    ids.push(created.id);
                    existingList.push(created);
                }
            }
            return ids;
        }

        if (Array.isArray(value)) {
            return value.map((e) => e.id);
        }

        return [];
    }

    async function upsertImportedMembers(imported: Member[]) {
        const existingMap = new Map(members.map(m => [m.email.toLowerCase(), m]));
        setImportInProgress(true);
        let i = 0;
        const max = imported.length;

        const [allRoles, allGroups, allSections] = await Promise.all([
            fetchRoles(),
            fetchGroups(),
            fetchSections(),
        ]);

        const rolesList = [...allRoles] as { id: number; name: string }[];
        const groupsList = [...allGroups] as { id: number; name: string }[];
        const sectionsList = [...allSections] as { id: number; name: string }[];

        for (const member of imported) {
            const emailKey = member.email.toLowerCase();
            const progress = Math.round((i / max) * 100);
            setImportProgress(progress);
            i++;

            const roleIds = await resolveEntityIds(
                member.roles as unknown as { id: number; name: string }[] | string | undefined,
                rolesList,
                (data) => createRole(data as unknown as Omit<Role, 'id'>)
            );
            const groupIds = await resolveEntityIds(
                member.groups as unknown as { id: number; name: string }[] | string | undefined,
                groupsList,
                (data) => createGroup(data as unknown as Omit<Group, 'id'>)
            );
            const sectionIds = await resolveEntityIds(
                member.sections as unknown as { id: number; name: string }[] | string | undefined,
                sectionsList,
                (data) => createSection(data as unknown as Omit<ClubSection, 'id'>)
            );

            if (existingMap.has(emailKey)) {
                const existing = existingMap.get(emailKey)!;
                // When a field is not mapped (undefined), preserve the existing associations
                const mergedRoleIds = member.roles !== undefined ? roleIds : (existing.roles?.map(r => r.id) || []);
                const mergedGroupIds = member.groups !== undefined ? groupIds : (existing.groups?.map(g => g.id) || []);
                const mergedSectionIds = member.sections !== undefined ? sectionIds : (existing.sections?.map(s => s.id) || []);
                const updated = {
                    ...existing,
                    ...member,
                    id: existing.id,
                    roleIds: mergedRoleIds,
                    groupIds: mergedGroupIds,
                    sectionIds: mergedSectionIds,
                };
                const saved = await updateMember(updated as Member);
                onMemberUpdated(saved);
            } else {
                const payload = { ...member, roleIds, groupIds, sectionIds };
                const saved = await createMember(payload as unknown as Omit<Member, 'id'>);
                onMemberUpdated(saved);
            }
        }
        setImportProgress(100);
        setImportInProgress(false);
    }

    const agGridThemeClass =
        theme.palette.mode === 'dark'
            ? 'ag-theme-clubmanager-dark'
            : 'ag-theme-clubmanager-light';

    return (
        <Box sx={{height: "calc(100vh - 300px)", width: "100%"}}>


            <Box display="flex" justifyContent="space-between" alignItems="center" mb={1}>
                <Typography variant="body1">{t("members.table.description")}</Typography>

                <Box display="flex" justifyContent="end">
                    <Tooltip title={t("tooltips.edit")}>
                        <IconButton onClick={handleEditMember} disabled={selectedMembers?.length !== 1} color={"primary"}>
                            <EditIcon/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.delete")}>
                        <IconButton onClick={handleDeleteMembers} disabled={!selectedMembers?.length} color="primary">
                            <Delete/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.export")}>
                        <IconButton onClick={handleExport} color={"primary"}>
                            <FileDownload/>
                        </IconButton>
                    </Tooltip>
                    <Tooltip title={t("tooltips.import")}>
                        <IconButton onClick={() => setImportWizardOpen(true)} color={"primary"}>
                            <FileUpload/>
                        </IconButton>
                    </Tooltip>

                    {importWizardOpen && (
                        <ImportMembersWizard
                            onClose={() => setImportWizardOpen(false)}
                            onImport={async (imported: Member[]) => {
                                setImportWizardOpen(false);
                                await upsertImportedMembers(imported);
                            }}
                        />
                    )}
                </Box>
            </Box>
            <Box sx={{height: "calc(100vh - 300px)", width: "100%", overflowX: "auto"}}>
                <div style={{minWidth: "1000px", height: "100%"}} className={agGridThemeClass}>

                    <DataGrid
                        autoHeight
                        rows={members}
                        columns={columnDefs.map((col) => ({
                            ...col,
                            sortable: true,
                            filterable: true,
                            minWidth: 150,
                        }))}
                        columnVisibilityModel={columnVisibility}
                        pageSizeOptions={[5, 10, 25]}
                        onRowSelectionModelChange={(selectionModel: GridRowSelectionModel) => {
                            handleSelectionChanged(selectionModel);
                        }}
                        onColumnVisibilityModelChange={(visibilityModel: GridColumnVisibilityModel) => {
                            handleColumnToggle(visibilityModel)
                        }}
                        checkboxSelection />

                </div>
            </Box>
            {editingMember && (
                <EditMemberDialog
                    member={editingMember!}
                    onClose={() => setEditingMember(null)}
                    onSave={async (update: Member) => {
                        await handleSaveEdit(update);
                    }}
                />
            )}
            {deletingMembers && (
                <DeletingMemberDialog
                    members={deletingMembers!}
                    onClose={() => setDeletingMembers(null)}
                    onDelete={(deleted: Member[]) => {
                        deleteMembers(deleted!).then(() => {
                            onMembersDeleted(deleted!);
                            setDeletingMembers(null)
                        })
                    }}/>
            )}
            {exportOpened && (
                <ExportMembersDialog members={members} onClose={() => setExportOpened(false)}/>
            )}
            <Backdrop
                sx={{
                    color: '#fff',
                    zIndex: (theme) => theme.zIndex.drawer + 1
                }}
                open={importInProgress}
            >
                <Box textAlign="center">
                    <CircularProgress color="inherit"/>
                    <Typography sx={{mt: 2}}>{t("members.table.importInProgress")} ({importProgress}%)</Typography>
                </Box>
            </Backdrop>
        </Box>
    );
}
