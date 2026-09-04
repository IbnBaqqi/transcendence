import type { AdminUserStatus, UserRole } from "../api/types";

const ROLES: readonly UserRole[] = ["USER", "ADMIN"];
const STATUSES: readonly AdminUserStatus[] = ["active", "suspended", "deleted"];

// The backend rejects anything past math.MaxInt32.
const MAX_PAGE = 2_147_483_647;

// "" is "no filter", as the URL and the selects both hold it.
export interface AdminUserFilters {
  role: UserRole | "";
  status: AdminUserStatus | "";
  page: number;
}

export const emptyAdminUserFilters: AdminUserFilters = { role: "", status: "", page: 1 };

// A URL is user input: ?status=banana and ?page=abc are coerced rather than
// forwarded, because the API answers both with a 400 and a 400 reads to the
// reader as "the server is down".
export function parseAdminUserFilters(sp: URLSearchParams): AdminUserFilters {
  const role = sp.get("role") as UserRole | null;
  const status = sp.get("status") as AdminUserStatus | null;
  const page = Number(sp.get("page"));

  return {
    role: role !== null && ROLES.includes(role) ? role : "",
    status: status !== null && STATUSES.includes(status) ? status : "",
    page: Number.isInteger(page) && page > 0 && page <= MAX_PAGE ? page : 1,
  };
}

// The only updater, so "change a filter, go back to page 1" lives in one place -
// otherwise filtering down to three accounts while on page 5 strands the reader
// on a valid empty page.
export function withAdminUserFilters(
  prev: AdminUserFilters,
  patch: Partial<AdminUserFilters>,
): AdminUserFilters {
  return { ...prev, ...patch, page: patch.page ?? 1 };
}

// Defaults stay out of the URL: an untouched list is /admin/users.
export function adminUserFiltersToParams(f: AdminUserFilters): URLSearchParams {
  const sp = new URLSearchParams();
  if (f.role !== "") sp.set("role", f.role);
  if (f.status !== "") sp.set("status", f.status);
  if (f.page !== 1) sp.set("page", String(f.page));
  return sp;
}

// Same string the request and the cache key are built from, so the two cannot
// describe different views.
export function adminUserFiltersToQuery(f: AdminUserFilters): string {
  return adminUserFiltersToParams(f).toString();
}
