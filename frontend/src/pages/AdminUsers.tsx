import { useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useAdminUsers } from "../api/adminUsers";
import { isApiError } from "../api/client";
import { Skeleton } from "../components/objects/Skeleton";
import {
  adminUserFiltersToParams,
  adminUserFiltersToQuery,
  parseAdminUserFilters,
  withAdminUserFilters,
  type AdminUserFilters,
} from "../lib/adminUserFilters";
import type { AdminUser } from "../api/types";

const STATUS_STYLES: Record<AdminUser["status"], string> = {
  active: "text-foreground",
  suspended: "text-berry-500",
  // Deleted rows carry no information but do take up space, so they stay
  // legible and quiet rather than being filtered out of the default view.
  deleted: "text-muted",
};

function UserRow({ user }: { user: AdminUser }) {
  const { t } = useTranslation();
  const deleted = user.status === "deleted";

  return (
    <li className="border-line bg-surface flex items-start justify-between gap-4 rounded-lg border p-3">
      <div className="min-w-0">
        {/* deleted-<id> is anonymisation, not a name: showing it would put an
            id on screen where a person used to be. */}
        <p className={`font-medium ${deleted ? "text-muted italic" : "text-foreground"}`}>
          {deleted ? t("adminUsers.deletedUser") : user.username}
        </p>
        {!deleted && <p className="text-muted text-sm">{user.email}</p>}
        {user.suspension_reason && (
          <p className="text-muted mt-1 text-sm">
            {t("adminUsers.suspendedFor", { reason: user.suspension_reason })}
          </p>
        )}
      </div>

      <div className="flex shrink-0 flex-col items-end gap-1 text-sm">
        <span className={STATUS_STYLES[user.status]}>{t(`adminUsers.status.${user.status}`)}</span>
        <span className="text-muted text-xs">{t(`adminUsers.role.${user.role}`)}</span>
        {/* A moderation signal, not presence: this ignores the user's own
            show_online_status setting, so it must never read as "online". */}
        <span className="text-muted text-xs">
          {user.last_seen_at
            ? t("adminUsers.lastSeen", { date: new Date(user.last_seen_at).toLocaleDateString() })
            : t("adminUsers.neverSeen")}
        </span>
      </div>
    </li>
  );
}

export default function AdminUsers() {
  const { t } = useTranslation();
  const [params, setParams] = useSearchParams();
  // Parsed every render rather than mirrored into state: the URL is the filters.
  const filters = parseAdminUserFilters(params);

  const update = (patch: Partial<AdminUserFilters>) =>
    setParams(adminUserFiltersToParams(withAdminUserFilters(filters, patch)));

  // The route is behind RequireAdmin, so isPending means loading here rather
  // than the "never ran" of a disabled query.
  const { data, isPending, isError, error, refetch } = useAdminUsers(
    adminUserFiltersToQuery(filters),
  );

  return (
    <div className="mx-auto max-w-4xl space-y-4 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.adminUsers.title")}</h1>

      {/* Two fixed lists with nothing to validate, so they commit on change -
          a submit button would be ceremony around a pair of selects. */}
      <div className="flex flex-wrap gap-3">
        <label className="text-muted text-sm">
          {t("adminUsers.filterRole")}
          <select
            value={filters.role}
            onChange={(event) => update({ role: event.target.value as AdminUserFilters["role"] })}
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          >
            <option value="">{t("adminUsers.anyRole")}</option>
            <option value="USER">{t("adminUsers.role.USER")}</option>
            <option value="ADMIN">{t("adminUsers.role.ADMIN")}</option>
          </select>
        </label>

        <label className="text-muted text-sm">
          {t("adminUsers.filterStatus")}
          <select
            value={filters.status}
            onChange={(event) =>
              update({ status: event.target.value as AdminUserFilters["status"] })
            }
            className="border-line bg-surface text-foreground ml-2 rounded border p-1 text-sm"
          >
            <option value="">{t("adminUsers.anyStatus")}</option>
            <option value="active">{t("adminUsers.status.active")}</option>
            <option value="suspended">{t("adminUsers.status.suspended")}</option>
            <option value="deleted">{t("adminUsers.status.deleted")}</option>
          </select>
        </label>
      </div>

      {isPending && (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      )}

      {isError && !data && (
        <Skeleton
          variant="error"
          className="h-24 w-full"
          message={isApiError(error) ? error.message : t("adminUsers.listError")}
          onRetry={() => refetch()}
        />
      )}

      {/* total, not the length of this page: an empty page past the end is not
          the same as nothing matching. */}
      {data?.total === 0 && (
        <div className="border-line rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">{t("adminUsers.listEmpty")}</p>
        </div>
      )}

      {data && data.items.length > 0 && (
        <ul className="space-y-3">
          {data.items.map((user) => (
            <UserRow key={user.id} user={user} />
          ))}
        </ul>
      )}
    </div>
  );
}
