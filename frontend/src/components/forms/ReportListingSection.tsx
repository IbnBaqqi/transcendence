import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useListing, useReportListing } from "../../api/listings";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import Button from "../objects/Button";
import type { ReportReason } from "../../api/types";

const REASONS: ReportReason[] = ["spam", "prohibited", "misleading", "offensive", "other"];

const DETAIL_MAX = 500;

export function ReportListingSection({ listingId }: { listingId: string }) {
  const { user } = useAuth();
  // By id rather than as a prop: React Query dedupes against the page's own
  // fetch, so this costs nothing and the section drops into #21's stub.
  const { data: listing } = useListing(listingId);

  // Signed out, still loading, or your own listing - the API refuses that last
  // one with a 400, so offering the control would be a guaranteed error.
  if (!listingId || !user || !listing || user.id === listing.seller_id) return null;

  return <ReportForm listingId={listingId} />;
}

function ReportForm({ listingId }: { listingId: string }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [detail, setDetail] = useState("");
  const report = useReportListing();

  const trimmed = detail.trim();
  // Mirrors the server: every other reason states the problem by itself, while
  // "other" says only that none of them fit.
  const detailRequired = reason === "other";
  const canSubmit = reason !== null && (!detailRequired || trimmed !== "");

  // One report per person per listing, so a second is a 409 - which means the
  // complaint is already on record. That is the outcome the reporter wanted,
  // so it ends the flow like a success instead of turning red.
  const alreadyFiled = isApiError(report.error) && report.error.status === 409;

  function submit() {
    if (!canSubmit || reason === null) return;
    report.mutate({ listingId, reason, ...(trimmed === "" ? {} : { detail: trimmed }) });
  }

  if (report.isSuccess || alreadyFiled) {
    return (
      <p className="text-muted text-sm">{t(alreadyFiled ? "report.already" : "report.thanks")}</p>
    );
  }

  if (!open) {
    return (
      <Button variant="tertiary" onClick={() => setOpen(true)}>
        {t("report.open")}
      </Button>
    );
  }

  return (
    <div className="border-line space-y-3 rounded-lg border p-4">
      <fieldset className="space-y-1">
        <legend className="text-muted text-sm">{t("report.legend")}</legend>
        {REASONS.map((option) => (
          <label key={option} className="text-foreground flex items-center gap-2 text-sm">
            <input
              type="radio"
              // One shared name is what makes these a group: arrow keys move
              // between them and only one can be checked.
              name={`report-reason-${listingId}`}
              value={option}
              checked={reason === option}
              onChange={() => setReason(option)}
            />
            {t(`moderation.reason.${option}`)}
          </label>
        ))}
      </fieldset>

      <div className="space-y-2">
        <label className="text-muted block text-sm" htmlFor={`report-detail-${listingId}`}>
          {detailRequired ? t("report.detailRequired") : t("report.detailOptional")}
        </label>
        <textarea
          id={`report-detail-${listingId}`}
          value={detail}
          onChange={(event) => setDetail(event.target.value)}
          // Mirrors the backend's own cap, so the field stops the reporter
          // rather than a round trip doing it.
          maxLength={DETAIL_MAX}
          rows={3}
          className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
        />
        <span className="text-muted text-xs">
          {t("moderation.noteRemaining", { count: DETAIL_MAX - detail.length })}
        </span>
      </div>

      <div className="flex items-center gap-2">
        <Button variant="primary" onClick={submit} disabled={!canSubmit || report.isPending}>
          {report.isPending ? t("report.submitting") : t("report.submit")}
        </Button>
        <Button variant="secondary" onClick={() => setOpen(false)}>
          {t("common.cancel")}
        </Button>
      </div>

      {report.isError && !alreadyFiled && (
        <p role="alert" className="text-danger text-sm">
          {isApiError(report.error) ? report.error.message : t("report.error")}
        </p>
      )}
    </div>
  );
}
