import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useTranslation } from "react-i18next";

import { Form } from "./Form";
import { FormField } from "./FormField";
import { useDeleteAccount } from "../../api/profile";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { makeDeleteAccountSchema, type DeleteAccountFormValues } from "../../schemas/deleteAccount";
import Button from "../objects/Button.tsx";

export function DeleteAccountSection({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { user, logout } = useAuth();
  const deleteAccount = useDeleteAccount();

  const username = user?.username ?? "";
  const form = useForm<DeleteAccountFormValues>({
    resolver: zodResolver(makeDeleteAccountSchema(username)),
    mode: "onChange",
    defaultValues: { confirmation: "" },
  });

  const handleSubmit = async (data: DeleteAccountFormValues) => {
    await deleteAccount.mutateAsync(data.confirmation);
    // The account is gone, so the session is too. logout swallows its own
    // server call failing - which it will, the session having just ended -
    // and clears the token, the user and every cached query.
    await logout();
    onClose();
  };

  return (
    <Form form={form} onSubmit={handleSubmit} isEditing={true}>
      <div className="space-y-4">
        {/* The row is anonymised, not removed: orders survive on both sides and
            the counterparty keeps their copy of a shared thread. Promising
            erasure would describe something the system does not do. */}
        <p className="text-muted text-sm">{t("modal.deleteAccount.warning")}</p>

        <FormField
          label={t("modal.deleteAccount.confirmLabel", { username })}
          name="confirmation"
          validateOnChange
        />

        {deleteAccount.isError && (
          <p role="alert" className="text-berry-500 text-sm">
            {isApiError(deleteAccount.error)
              ? deleteAccount.error.message
              : t("modal.deleteAccount.failed")}
          </p>
        )}

        <div className="flex flex-row gap-2">
          <Button
            variant="secondary"
            type="submit"
            disabled={!form.formState.isValid || deleteAccount.isPending}
          >
            {deleteAccount.isPending ? t("common.deleting") : t("common.deleteAccount")}
          </Button>
          <Button variant="primary" type="button" onClick={onClose}>
            {t("common.cancel")}
          </Button>
        </div>
      </div>
    </Form>
  );
}
