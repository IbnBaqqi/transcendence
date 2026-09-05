import { useState } from "react";
import { useTranslation } from "react-i18next";

import {
  useDeleteUser,
  useReinstateUser,
  useSetUserRole,
  useSuspendUser,
} from "../../api/adminUsers";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import Button from "./Button";
import type { AdminUser } from "../../api/types";

const REASON_MAX = 500;

export function AccountActions({ account }: { account: AdminUser }) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [mode, setMode] = useState<null | "primary" | "role" | "delete">(null);
  const [reason, setReason] = useState("");
  const [typedName, setTypedName] = useState("");

  const suspend = useSuspendUser();
  const reinstate = useReinstateUser();
  const remove = useDeleteUser();
  const setRole = useSetUserRole();

  // Deletion is the one state with no way back, and an admin aiming at their
  // own account only ever gets a 403 - suspending yourself would lock you out
  // of the endpoint that undoes it. Neither is an error to report; both are
  // buttons not to offer.
  if (account.status === "deleted" || user?.id === account.id) return null;

  const suspending = account.status === "active";
  const promoting = account.role !== "ADMIN";
  const pending = suspend.isPending || reinstate.isPending || remove.isPending || setRole.isPending;
  const error = suspend.error ?? reinstate.error ?? remove.error ?? setRole.error;
  const failed = suspend.isError || reinstate.isError || remove.isError || setRole.isError;

  // Required for a suspension alone, because the API requires it there alone.
  const trimmed = reason.trim();
  const canSubmit = !suspending || trimmed !== "";

  // Exact, because the API means exact. Trimmed on both sides of the
  // comparison and in the request, so the check and the body cannot disagree -
  // and a username cannot contain surrounding whitespace anyway.
  const typed = typedName.trim();
  const canDelete = typed === account.username && trimmed !== "";

  const done = () => {
    setMode(null);
    setReason("");
    setTypedName("");
    // The error is derived from the mutations, so clearing only the form
    // leaves a red 409 under a collapsed row with nothing to explain it - and
    // onError refetches, so the row can redraw as Suspended while the line
    // beneath still says the suspend failed.
    suspend.reset();
    reinstate.reset();
    remove.reset();
    setRole.reset();
  };

  function submit() {
    if (!canSubmit) return;

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

  function submitRole() {
    // Omitted rather than sent as "" when blank, like the reinstate note: the
    // API calls it optional, and an empty string is a value the audit trail
    // would keep.
    setRole.mutate(
      {
        userId: account.id,
        role: promoting ? "ADMIN" : "USER",
        ...(trimmed === "" ? {} : { note: trimmed }),
      },
      { onSuccess: done },
    );
  }

  function submitDelete() {
    if (!canDelete) return;
    remove.mutate({ userId: account.id, username: typed, reason: trimmed }, { onSuccess: done });
  }

  return (
    <div className="flex flex-col items-end gap-2">
      {mode === null && (
        // Wraps, like the other button rows: three actions do not fit a phone,
        // and the longest labels are Finnish and Swedish rather than English -
        // "Poista yllapito-oikeudet" is what tips it at 390px.
        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="secondary" onClick={() => setMode("primary")}>
            {t(suspending ? "adminUsers.suspend" : "adminUsers.reinstate")}
          </Button>
          <Button variant="secondary" onClick={() => setMode("role")}>
            {t(promoting ? "adminUsers.makeAdmin" : "adminUsers.removeAdmin")}
          </Button>
          <Button variant="tertiary" onClick={() => setMode("delete")}>
            {t("adminUsers.delete")}
          </Button>
        </div>
      )}

      {mode === "primary" && (
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
            <Button variant="tertiary" onClick={done}>
              {t("common.cancel")}
            </Button>
            <Button variant="primary" onClick={submit} disabled={!canSubmit || pending}>
              {pending ? t("adminUsers.working") : t("adminUsers.confirm")}
            </Button>
          </div>
        </div>
      )}

      {mode === "role" && (
        <div className="w-64 space-y-2">
          {/* Confirmed, but not by typing a name. Deletion asks for that
              because it cannot be undone; a demotion is undone by the same
              control that did it, and over-confirming a reversible action
              trains people to click through the one that is not. */}
          <p className="text-muted text-sm">
            {t(promoting ? "adminUsers.makeAdminWarning" : "adminUsers.removeAdminWarning", {
              username: account.username,
            })}
          </p>

          <label className="text-muted block text-sm" htmlFor={`role-note-${account.id}`}>
            {t("adminUsers.noteOptional")}
          </label>
          <textarea
            id={`role-note-${account.id}`}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={REASON_MAX}
            rows={2}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />

          <div className="flex justify-end gap-2">
            <Button variant="tertiary" onClick={done}>
              {t("common.cancel")}
            </Button>
            <Button variant="primary" onClick={submitRole} disabled={pending}>
              {pending ? t("adminUsers.working") : t("adminUsers.confirm")}
            </Button>
          </div>
        </div>
      )}

      {mode === "delete" && (
        <div className="w-64 space-y-2">
          {/* Anonymised, not erased: orders survive on both sides and the
              counterparty keeps their copy of a shared thread. Promising
              removal here would describe something the system does not do. */}
          <p className="text-muted text-sm">{t("adminUsers.deleteWarning")}</p>

          <label className="text-muted block text-sm" htmlFor={`confirm-${account.id}`}>
            {/* Naming the account is the guard: in a list of adjacent rows a
                yes/no dialog confirms the act, never the target. */}
            {t("adminUsers.confirmUsername", { username: account.username })}
          </label>
          <input
            id={`confirm-${account.id}`}
            value={typedName}
            onChange={(event) => setTypedName(event.target.value)}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />

          <label className="text-muted block text-sm" htmlFor={`delete-reason-${account.id}`}>
            {t("adminUsers.deleteReason")}
          </label>
          <textarea
            id={`delete-reason-${account.id}`}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={REASON_MAX}
            rows={2}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />

          <div className="flex justify-end gap-2">
            <Button variant="tertiary" onClick={done}>
              {t("common.cancel")}
            </Button>
            <Button variant="secondary" onClick={submitDelete} disabled={!canDelete || pending}>
              {pending ? t("adminUsers.working") : t("adminUsers.delete")}
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
