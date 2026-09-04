import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useAdminOrders } from "../api/adminOrders";
import { isApiError } from "../api/client";
import { AdminOrderRow } from "../components/objects/AdminOrderRow";
import { Pagination } from "../components/objects/Pagination";
import { Skeleton } from "../components/objects/Skeleton";
import {
  adminOrderFiltersToParams,
  adminOrderFiltersToQuery,
  parseAdminOrderFilters,
  rangeIsBackwards,
  withAdminOrderFilters,
  type AdminOrderFilters,
} from "../lib/adminOrderFilters";
import type { OrderStatus } from "../api/types";

const STATUSES: OrderStatus[] = ["pending", "confirmed", "completed", "cancelled", "refunded"];

export default function AdminOrders() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  // Parsed every render rather than mirrored into state: the URL is the filters.
  const filters = parseAdminOrderFilters(params);

  const update = (patch: Partial<AdminOrderFilters>) =>
    setParams(adminOrderFiltersToParams(withAdminOrderFilters(filters, patch)));

  // The API answers a reversed range with a 400, which reads as "the server is
  // down" rather than "those dates are the wrong way round".
  const backwards = rangeIsBackwards(filters);

  // Two serialisers, and they are not interchangeable: ToParams writes the URL
  // in the dates the admin picked, ToQuery converts them to the instants the
  // API means. See lib/adminOrderFilters.ts.
  const { data, isPending, isError, error, refetch } = useAdminOrders(
    adminOrderFiltersToQuery(filters),
  );

  return (
    <div className="mx-auto max-w-4xl space-y-4 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.adminOrders.title")}</h1>

      <div className="flex flex-wrap items-end gap-3">
        <label className="text-muted text-sm">
          {t("adminOrders.filterStatus")}
          <select
            value={filters.status}
            onChange={(event) =>
              update({ status: event.target.value as AdminOrderFilters["status"] })
            }
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          >
            <option value="">{t("adminOrders.anyStatus")}</option>
            {STATUSES.map((status) => (
              <option key={status} value={status}>
                {t(`orders.status.${status}`)}
              </option>
            ))}
          </select>
        </label>

        {/* Three states, not a checkbox: the API distinguishes "either" from
            "only the ones that are not stuck". */}
        <label className="text-muted text-sm">
          {t("adminOrders.filterStuck")}
          <select
            value={filters.stuck}
            onChange={(event) =>
              update({ stuck: event.target.value as AdminOrderFilters["stuck"] })
            }
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          >
            <option value="">{t("adminOrders.anyStuck")}</option>
            <option value="true">{t("adminOrders.onlyStuck")}</option>
            <option value="false">{t("adminOrders.notStuck")}</option>
          </select>
        </label>

        {/* Inclusive at both ends, which is what the inputs now mean - the
            exclusive upper bound is applied on the way to the API. */}
        <label className="text-muted text-sm">
          {t("adminOrders.from")}
          <input
            type="date"
            value={filters.created_from}
            onChange={(event) => update({ created_from: event.target.value })}
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          />
        </label>

        <label className="text-muted text-sm">
          {t("adminOrders.to")}
          <input
            type="date"
            value={filters.created_to}
            onChange={(event) => update({ created_to: event.target.value })}
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          />
        </label>
      </div>

      {backwards && (
        <p role="alert" className="text-berry-500 text-sm">
          {t("adminOrders.rangeBackwards")}
        </p>
      )}

      {isPending && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 w-full" />
          ))}
        </div>
      )}

      {isError && !data && (
        <Skeleton
          variant="error"
          className="h-24 w-full"
          message={isApiError(error) ? error.message : t("adminOrders.listError")}
          onRetry={() => refetch()}
        />
      )}

      {/* total, not the length of this page: an empty page past the end is not
          the same as nothing matching. */}
      {data?.total === 0 && (
        <div className="border-line rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">{t("adminOrders.listEmpty")}</p>
        </div>
      )}

      {data && data.items.length > 0 && (
        <ul className="space-y-3">
          {data.items.map((order) => (
            <AdminOrderRow key={order.id} order={order} />
          ))}
        </ul>
      )}

      {data && data.total > 0 && (
        <Pagination
          page={data.page}
          totalPages={data.total_pages}
          total={data.total}
          onPageChange={(page) => update({ page })}
        />
      )}
    </div>
  );
}
