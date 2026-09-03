import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useListingReports, useModerationHistory } from "../../api/moderation";
import type { ModerationAction, Report, ReportedListing } from "../../api/types";

function ReportLine({ report }: { report: Report }) {
  const { t } = useTranslation();
  const open = report.status === "open";

  return (
    <li className={`py-2 ${open ? "" : "opacity-60"}`}>
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-foreground text-sm font-medium">
          {t(`moderation.reason.${report.reason}`)}
        </span>
        <span className="text-muted shrink-0 text-xs">
          {t(`moderation.status.${report.status}`)} ·{" "}
          {new Date(report.created_at).toLocaleDateString()}
        </span>
      </div>
      {/* Attacker-controlled text. React escapes it; never render it as markup. */}
      {report.detail && <p className="text-muted mt-0.5 text-sm">{report.detail}</p>}
      {report.reporter_id === null && (
        <p className="text-muted mt-0.5 text-xs italic">{t("moderation.deletedReporter")}</p>
      )}
    </li>
  );
}

function HistoryLine({ entry }: { entry: ModerationAction }) {
  const { t } = useTranslation();

  return (
    <li className="py-2">
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-foreground text-sm font-medium">
          {t(`moderation.action.${entry.action}`)}
        </span>
        <span className="text-muted shrink-0 text-xs">
          {new Date(entry.created_at).toLocaleDateString()}
        </span>
      </div>
      {entry.note && <p className="text-muted mt-0.5 text-sm">{entry.note}</p>}
      {entry.moderator_id === null && (
        <p className="text-muted mt-0.5 text-xs italic">{t("moderation.deletedModerator")}</p>
      )}
    </li>
  );
}

export function ReportedListingRow({ row }: { row: ReportedListing }) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const removed = row.removed_at !== null;

  // A collapsed row passes "", which leaves both queries disabled - so opening
  // one row is what costs two requests, not rendering the queue.
  const listingId = expanded ? row.listing_id : "";
  const { data: reports } = useListingReports(listingId);
  const { data: history } = useModerationHistory(listingId);

  return (
    <li className="border-line bg-surface rounded-lg border p-3">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <Link
            to={`/listings/${row.listing_id}`}
            className={`font-medium hover:underline ${removed ? "text-muted line-through" : "text-foreground"}`}
          >
            {row.title}
          </Link>
          <p className="text-muted mt-1 text-sm">
            {t("moderation.firstReported", {
              date: new Date(row.first_reported_at).toLocaleDateString(),
            })}
          </p>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1">
          {/* The count leads: three complaints about one listing is one problem,
              and it is what decides which row to open first. */}
          <span className="bg-accent text-accent-contrast rounded-full px-2 py-0.5 text-xs font-medium">
            {t("moderation.reportCount", { count: row.report_count })}
          </span>
          {removed && <span className="text-muted text-xs">{t("moderation.alreadyRemoved")}</span>}
          <button
            type="button"
            onClick={() => setExpanded((value) => !value)}
            aria-expanded={expanded}
            className="text-accent cursor-pointer text-xs"
          >
            {expanded ? t("moderation.hideDetail") : t("moderation.showDetail")}
          </button>
        </div>
      </div>

      {expanded && (
        <div className="border-line mt-3 grid gap-4 border-t pt-3 sm:grid-cols-2">
          <section>
            <h3 className="text-foreground text-sm font-bold">{t("moderation.reportsTitle")}</h3>
            {/* Resolved reports stay: a listing reported before and cleared is
                context for deciding again. Server order, newest first. */}
            {reports && reports.length > 0 ? (
              <ul className="divide-line divide-y">
                {reports.map((report) => (
                  <ReportLine key={report.id} report={report} />
                ))}
              </ul>
            ) : (
              <p className="text-muted mt-1 text-sm">
                {reports ? t("moderation.reportsEmpty") : t("common.loading")}
              </p>
            )}
          </section>

          <section>
            <h3 className="text-foreground text-sm font-bold">{t("moderation.historyTitle")}</h3>
            {history && history.length > 0 ? (
              <ul className="divide-line divide-y">
                {history.map((entry) => (
                  <HistoryLine key={entry.id} entry={entry} />
                ))}
              </ul>
            ) : (
              <p className="text-muted mt-1 text-sm">
                {history ? t("moderation.historyEmpty") : t("common.loading")}
              </p>
            )}
          </section>
        </div>
      )}
    </li>
  );
}
