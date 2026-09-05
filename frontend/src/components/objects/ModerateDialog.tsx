import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useModerateListing } from "../../api/moderation";
import { isApiError } from "../../api/client";
import Button from "./Button";
import type { ModerationRequestAction } from "../../api/types";

const NOTE_MAX = 500;

export function ModerateDialog({ listingId, removed }: { listingId: string; removed: boolean }) {
  const { t } = useTranslation();
  const [action, setAction] = useState<ModerationRequestAction | null>(null);
  const [note, setNote] = useState("");
  const moderate = useModerateListing();

  // removed_at decides what is even offered. Showing all three regardless is
  // how you generate 409s on purpose: the API refuses a remove on an already
  // removed listing and a restore on one that is not.
  const actions: ModerationRequestAction[] = removed ? ["restore"] : ["remove", "dismiss"];

  // The API requires a reason for remove alone - an audit trail whose reason
  // is empty is only a timestamp.
  const noteRequired = action === "remove";
  const trimmed = note.trim();
  const canSubmit = action !== null && (!noteRequired || trimmed !== "");

  function submit() {
    if (!canSubmit || action === null) return;
    moderate.mutate(
      // Omitted rather than "" when blank: the API calls note optional, and an
      // empty string is a value that would land in the audit trail as one.
      { listingId, action, ...(trimmed === "" ? {} : { note: trimmed }) },
      {
        // The row leaves the queue on its own - the mutation invalidates it.
        onSuccess: () => {
          setAction(null);
          setNote("");
        },
      },
    );
  }

  return (
    <div className="border-line mt-3 border-t pt-3">
      <div className="flex flex-wrap items-center gap-2">
        {actions.map((option) => (
          <Button
            key={option}
            variant={action === option ? "primary" : "secondary"}
            onClick={() => setAction(option)}
          >
            {t(`moderation.decide.${option}`)}
          </Button>
        ))}
      </div>

      {action !== null && (
        <div className="mt-3 space-y-2">
          <label className="text-muted block text-sm" htmlFor={`note-${listingId}`}>
            {noteRequired ? t("moderation.noteRequired") : t("moderation.noteOptional")}
          </label>
          <textarea
            id={`note-${listingId}`}
            value={note}
            onChange={(event) => setNote(event.target.value)}
            // Mirrors the backend's own cap, so the field stops the moderator
            // rather than a round trip doing it.
            maxLength={NOTE_MAX}
            rows={2}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted text-xs">
              {t("moderation.noteRemaining", { count: NOTE_MAX - note.length })}
            </span>
            <Button variant="primary" onClick={submit} disabled={!canSubmit || moderate.isPending}>
              {moderate.isPending ? t("moderation.deciding") : t("moderation.confirm")}
            </Button>
          </div>
        </div>
      )}

      {/* Every failure here means this copy of the queue is stale - another
          moderator got there first, or the listing is gone. So the message and
          a refetch, rather than a branch per status. */}
      {moderate.isError && (
        <p role="alert" className="text-danger mt-2 text-sm">
          {isApiError(moderate.error) ? moderate.error.message : t("moderation.decideFailed")}
        </p>
      )}

      {moderate.isSuccess && moderate.data && (
        <p className="text-muted mt-2 text-sm">
          {t("moderation.resolvedCount", { count: moderate.data.reports_resolved })}
        </p>
      )}
    </div>
  );
}
