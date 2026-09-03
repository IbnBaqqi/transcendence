import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useReportQueue } from "../api/moderation";
import { isApiError } from "../api/client";
import { Skeleton } from "../components/objects/Skeleton";
import type { ReportedListing } from "../api/types";

function QueueRow({ row }: { row: ReportedListing }) {
  const { t } = useTranslation();
  const removed = row.removed_at !== null;

  return (
    <li className="border-line bg-surface flex items-start justify-between gap-4 rounded-lg border p-3">
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
      </div>
    </li>
  );
}

export default function AdminListings() {
  const { t } = useTranslation();

  // The route is behind RequireAdmin, so there is no signed-out branch here
  // and isPending means loading rather than "never ran".
  const { data: queue, isPending, isError, error, refetch } = useReportQueue();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.adminListings.title")}</h1>

      {isPending && (
        <div className="mt-6 space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      )}

      {isError && !queue && (
        <Skeleton
          variant="error"
          className="mt-6 h-24 w-full"
          message={isApiError(error) ? error.message : t("moderation.queueError")}
          onRetry={() => refetch()}
        />
      )}

      {queue?.length === 0 && (
        <div className="border-line mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">{t("moderation.queueEmpty")}</p>
        </div>
      )}

      {queue && queue.length > 0 && (
        // Oldest complaint first, which is the order the server already
        // returns - ORDER BY min(created_at). Sorting again here would be a
        // second opinion about the queue's priority.
        <ul className="mt-6 space-y-3">
          {queue.map((row) => (
            <QueueRow key={row.listing_id} row={row} />
          ))}
        </ul>
      )}
    </div>
  );
}
