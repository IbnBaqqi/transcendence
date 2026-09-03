import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useReinstateUser, useSuspendUser } from "../../api/adminUsers";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import Button from "./Button";
import type { AdminUser } from "../../api/types";

const REASON_MAX = 500;

export function AccountActions({ account }: { account: AdminUser }) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");

  const suspend = useSuspendUser();
  const reinstate = useReinstateUser();

  // Deletion is the one state with no way back, and an admin aiming at their
  // own account only ever gets a 403 - suspending yourself would lock you out
  // of the endpoint that undoes it. Neither is an error to report; both are
  // buttons not to offer.
  if (account.status === "deleted" || user?.id === account.id) return null;

  const suspending = account.status === "active";
  const pending = suspend.isPending || reinstate.isPending;
  const error = suspend.error ?? reinstate.error;
  const failed = suspend.isError || reinstate.isError;

  // Required for a suspension alone, because the API requires it there alone.
  const trimmed = reason.trim();
  const canSubmit = !suspending || trimmed !== "";

  function submit() {
    if (!canSubmit) return;
    const done = () => {
      setOpen(false);
      setReason("");
    };

    if (suspending) {
      suspend.mutate({ userId: account.id, reason: trimmed }, { onSuccess: done });
    } else {
      // Omitted rather than sent as "" when blank: the API calls the note
      // optional, and an empty string is a value the audit trail would keep.
      reinstate.mutate(
        { userId: account.id, ...(trimmed === "" ? {} : { note: trimmed }) },
        { onSuccess: done },
      );
    }
  }

  return (
    <div className="flex flex-col items-end gap-2">
      {!open ? (
        <Button variant="secondary" onClick={() => setOpen(true)}>
          {t(suspending ? "adminUsers.suspend" : "adminUsers.reinstate")}
        </Button>
      ) : (
        <div className="w-64 space-y-2">
          <label className="text-muted block text-sm" htmlFor={`reason-${account.id}`}>
            {/* The suspension reason is shown to the suspended user on their
                next request, so the label says who reads it - otherwise it
                gets written as a private note and they get one word. */}
            {t(suspending ? "adminUsers.reasonForUser" : "adminUsers.noteOptional")}
          </label>
          <textarea
            id={`reason-${account.id}`}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={REASON_MAX}
            rows={2}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />
          <div className="flex justify-end gap-2">
            <Button variant="tertiary" onClick={() => setOpen(false)}>
              {t("common.cancel")}
            </Button>
            <Button variant="primary" onClick={submit} disabled={!canSubmit || pending}>
              {pending ? t("adminUsers.working") : t("adminUsers.confirm")}
            </Button>
          </div>
        </div>
      )}

      {/* A 409 is another admin getting there first, or the last active admin -
          only the server's own message tells them apart. The list refetches
          either way; that is in the mutation. */}
      {failed && (
        <p role="alert" className="text-berry-500 text-sm">
          {isApiError(error) ? error.message : t("adminUsers.actionFailed")}
        </p>
      )}
    </div>
  );
}
