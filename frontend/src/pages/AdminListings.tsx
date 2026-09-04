import { useTranslation } from "react-i18next";

import { useReportQueue } from "../api/moderation";
import { isApiError } from "../api/client";
import { ReportedListingRow } from "../components/objects/ReportedListingRow";
import { Skeleton } from "../components/objects/Skeleton";

export default function AdminListings() {
  const { t } = useTranslation();

  // The route is behind RequireAdmin, so there is no signed-out branch here
  // and isPending means loading rather than "never ran".
  const { data: queue, isPending, isError, error, refetch } = useReportQueue();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">
        {t("pages.adminListings.title")}
      </h1>

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
            <ReportedListingRow key={row.listing_id} row={row} />
          ))}
        </ul>
      )}
    </div>
  );
}
