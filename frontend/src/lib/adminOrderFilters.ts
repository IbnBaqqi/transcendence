import type { OrderStatus } from "../api/types";

const STATUSES: readonly OrderStatus[] = [
  "pending",
  "confirmed",
  "completed",
  "cancelled",
  "refunded",
];

// Three states, which is why this is not a boolean: omitted means either,
// "true" means only stuck, "false" means only not.
const STUCK_VALUES = ["", "true", "false"] as const;
export type StuckFilter = (typeof STUCK_VALUES)[number];

const MAX_PAGE = 2_147_483_647;
const DAY_PATTERN = /^\d{4}-\d{2}-\d{2}$/;

// Dates are held as the admin picked them: YYYY-MM-DD, local, and created_to
// inclusive. The exclusive bound and the timezone are the API's business and
// are applied in toQuery, so a shared URL reads like the range somebody chose.
export interface AdminOrderFilters {
  status: OrderStatus | "";
  stuck: StuckFilter;
  created_from: string;
  created_to: string;
  page: number;
}

export const emptyAdminOrderFilters: AdminOrderFilters = {
  status: "",
  stuck: "",
  created_from: "",
  created_to: "",
  page: 1,
};

function parseDay(raw: string | null): string {
  if (raw === null || !DAY_PATTERN.test(raw)) return "";
  // Rejects 2026-02-31: Date rolls it forward to March, so a round trip that
  // does not come back is how an impossible day is caught.
  const [y, m, d] = raw.split("-").map(Number);
  const asDate = new Date(y, m - 1, d);
  const sameDay =
    asDate.getFullYear() === y && asDate.getMonth() === m - 1 && asDate.getDate() === d;
  return sameDay ? raw : "";
}

// A URL is user input: anything outside the enums is coerced rather than
// forwarded, because the API answers each with a 400 and a 400 reads as "the
// server is down" rather than "you typed that".
export function parseAdminOrderFilters(sp: URLSearchParams): AdminOrderFilters {
  const status = sp.get("status") as OrderStatus | null;
  const stuck = sp.get("stuck") as StuckFilter | null;
  const page = Number(sp.get("page"));

  return {
    status: status !== null && STATUSES.includes(status) ? status : "",
    stuck: stuck !== null && STUCK_VALUES.includes(stuck) ? stuck : "",
    created_from: parseDay(sp.get("created_from")),
    created_to: parseDay(sp.get("created_to")),
    page: Number.isInteger(page) && page > 0 && page <= MAX_PAGE ? page : 1,
  };
}

// The only updater, so "change a filter, go back to page 1" lives in one place.
export function withAdminOrderFilters(
  prev: AdminOrderFilters,
  patch: Partial<AdminOrderFilters>,
): AdminOrderFilters {
  return { ...prev, ...patch, page: patch.page ?? 1 };
}

// Defaults stay out of the URL: an untouched list is /admin/orders.
export function adminOrderFiltersToParams(f: AdminOrderFilters): URLSearchParams {
  const sp = new URLSearchParams();
  if (f.status !== "") sp.set("status", f.status);
  if (f.stuck !== "") sp.set("stuck", f.stuck);
  if (f.created_from !== "") sp.set("created_from", f.created_from);
  if (f.created_to !== "") sp.set("created_to", f.created_to);
  if (f.page !== 1) sp.set("page", String(f.page));
  return sp;
}

// The range ends before it starts - the API answers that with a 400, and the
// comparison is safe on the raw strings because YYYY-MM-DD sorts as dates do.
export function rangeIsBackwards(f: AdminOrderFilters): boolean {
  return f.created_from !== "" && f.created_to !== "" && f.created_from > f.created_to;
}

// Local midnight. new Date("2026-08-01") would parse as UTC and reintroduce
// the offset this whole conversion exists to remove.
function localStartOfDay(day: string, addDays = 0): Date {
  const [y, m, d] = day.split("-").map(Number);
  return new Date(y, m - 1, d + addDays);
}

// The same instant, as UTC - toISOString emits a Z, not a local offset. What
// matters is that it is an instant at all: parseDayBound tries RFC 3339 before
// the bare-date form, and sending YYYY-MM-DD would be read as UTC midnight, so
// from a zone ahead of UTC the range would end early.
function toInstant(day: string, addDays = 0): string {
  return localStartOfDay(day, addDays).toISOString();
}

export function adminOrderFiltersToQuery(f: AdminOrderFilters): string {
  const sp = new URLSearchParams();
  if (f.status !== "") sp.set("status", f.status);
  if (f.stuck !== "") sp.set("stuck", f.stuck);
  if (f.created_from !== "") sp.set("created_from", toInstant(f.created_from));
  // created_to is exclusive, so an inclusive 31 August is sent as the start of
  // 1 September - otherwise the last day the admin picked is left out.
  if (f.created_to !== "") sp.set("created_to", toInstant(f.created_to, 1));
  if (f.page !== 1) sp.set("page", String(f.page));
  return sp.toString();
}
