import {Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle,} from "@mui/material";
import {useTranslation} from "react-i18next";
import {Member} from "../../api/types";

type Props = {
    members: Member[];
    onClose: () => void;
    onDelete: (deleted: Member[]) => void;
};

export function DeletingMemberDialog({ members, onClose, onDelete }: Props) {
    const { t } = useTranslation();

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>
                {members.length === 1
                    ? t("members.dialogs.delete.title")
                    : t("members.dialogs.delete.titleMultiple")}
            </DialogTitle>
            <DialogContent>
            <DialogContentText>
                {members.length === 1
                    ? t("members.dialogs.delete.description")
                    : t("members.dialogs.delete.descriptionMultiple")}
            </DialogContentText>
                <ul>
                    {members.map((member) => (
                        <li key={member.id}>
                            {member.firstName} {member.lastName} ({member.email})
                        </li>
                    ))}
                </ul>
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button
                    variant="contained"
                    onClick={() => {
                        onDelete(members);
                    }}
                >
                    {t("buttons.delete")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
