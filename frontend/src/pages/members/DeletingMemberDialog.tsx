import {Button, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle,} from "@mui/material";
import {useTranslation} from "react-i18next";
import {Member} from "../../api/types";

type Props = {
    member: Member;
    onClose: () => void;
    onDelete: (deleted: Member) => void;
};

export function DeletingMemberDialog({ member, onClose, onDelete }: Props) {
    const { t } = useTranslation();

    return (
        <Dialog open onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>
                {t("members.dialogs.delete.title")}
            </DialogTitle>
            <DialogContent>
             <DialogContentText>{t("members.dialogs.delete.description")}</DialogContentText>
            </DialogContent>

            <DialogActions>
                <Button onClick={onClose}>{t("buttons.abort")}</Button>
                <Button
                    variant="contained"
                    onClick={() => {
                        onDelete(member);
                    }}
                >
                    {t("buttons.delete")}
                </Button>
            </DialogActions>
        </Dialog>
    );
}
