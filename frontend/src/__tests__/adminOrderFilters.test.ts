// @vitest-environment jsdom

// Pinned, and not to UTC. The local-vs-UTC midnight distinction this module
// exists for is invisible from UTC - the two are the same instant - so under
// the default runner zone (UTC in CI) the mutation that reintroduces the bug
// passes every test. Set before any Date is constructed.
// tsconfig.app.json lists only vite/client and vitest/globals, so `process` is
// untyped in app code - hence reaching it through globalThis rather than
// widening the project's global types for one test file.
(globalThis as unknown as { process: { env: Record<string, string> } }).process.env.TZ =
  "Europe/Helsinki";

import {
  adminOrderFiltersToParams,
  adminOrderFiltersToQuery,
  emptyAdminOrderFilters,
  parseAdminOrderFilters,
  rangeIsBackwards,
  withAdminOrderFilters,
} from "../lib/adminOrderFilters";

const parse = (query: string) => parseAdminOrderFilters(new URLSearchParams(query));
const query = (over: Partial<Parameters<typeof adminOrderFiltersToQuery>[0]> = {}) =>
  new URLSearchParams(adminOrderFiltersToQuery({ ...emptyAdminOrderFilters, ...over }));

describe("parsing", () => {
  test("reads the filters it recognises", () => {
    expect(parse("status=confirmed&stuck=true&created_from=2026-08-01&page=3")).toEqual({
      status: "confirmed",
      stuck: "true",
      created_from: "2026-08-01",
      created_to: "",
      page: 3,
    });
  });

  test("an empty query is the default view", () => {
    expect(parse("")).toEqual(emptyAdminOrderFilters);
  });

  // stuck has three states, which is why it is not a boolean: omitted means
  // either, and "false" means only orders that are not stuck.
  test.each([
    ["stuck=true", "true"],
    ["stuck=false", "false"],
    ["", ""],
  ] as const)("%s keeps its meaning", (q, expected) => {
    expect(parse(q).stuck).toBe(expected);
  });

  test.each([
    ["status=banana", "status"],
    ["status=Pending", "status"],
    ["stuck=banana", "stuck"],
    ["stuck=1", "stuck"],
  ] as const)("%s is dropped rather than forwarded", (q, field) => {
    expect(parse(q)[field]).toBe("");
  });

  test.each(["created_from=yesterday", "created_from=01-08-2026", "created_from=2026-8-1"])(
    "%s is not a date and is dropped",
    (q) => {
      expect(parse(q).created_from).toBe("");
    },
  );

  // Date rolls 31 February forward into March, so a day that does not survive
  // the round trip never existed.
  test("an impossible day is dropped", () => {
    expect(parse("created_from=2026-02-31").created_from).toBe("");
    expect(parse("created_from=2026-02-28").created_from).toBe("2026-02-28");
  });
});

describe("updating and serialising", () => {
  test("changing a filter returns to page 1", () => {
    const deep = { ...emptyAdminOrderFilters, status: "confirmed" as const, page: 5 };
    expect(withAdminOrderFilters(deep, { stuck: "true" }).page).toBe(1);
  });

  test("changing the page keeps the page", () => {
    expect(withAdminOrderFilters(emptyAdminOrderFilters, { page: 4 }).page).toBe(4);
  });

  test("an untouched list has no query string at all", () => {
    expect(adminOrderFiltersToParams(emptyAdminOrderFilters).toString()).toBe("");
  });

  // The URL holds what the admin picked - inclusive, local, plain - so a
  // shared link reads like the range somebody chose.
  test("the URL keeps the dates as they were picked", () => {
    const params = adminOrderFiltersToParams({
      ...emptyAdminOrderFilters,
      created_from: "2026-08-01",
      created_to: "2026-08-31",
    });
    expect(params.get("created_from")).toBe("2026-08-01");
    expect(params.get("created_to")).toBe("2026-08-31");
  });

  test("the range ends before it starts", () => {
    const backwards = {
      ...emptyAdminOrderFilters,
      created_from: "2026-08-31",
      created_to: "2026-08-01",
    };
    expect(rangeIsBackwards(backwards)).toBe(true);
    expect(rangeIsBackwards({ ...backwards, created_to: "2026-08-31" })).toBe(false);
    expect(rangeIsBackwards({ ...emptyAdminOrderFilters, created_from: "2026-08-31" })).toBe(false);
  });
});

describe("what actually reaches the API", () => {
  // created_to is exclusive, so including the whole of 31 August means asking
  // for everything before 1 September. Sending the 31st drops that day.
  test("the upper bound is sent as the day after the one picked", () => {
    const sent = query({ created_from: "2026-08-01", created_to: "2026-08-31" });
    expect(sent.get("created_to")).toBe(new Date(2026, 8, 1).toISOString());
    expect(sent.get("created_from")).toBe(new Date(2026, 7, 1).toISOString());
  });

  // The instant is local midnight, not UTC midnight. new Date("2026-08-01")
  // would be UTC and reintroduce the offset this conversion exists to remove -
  // so under a zone ahead of UTC the two differ, and that is the assertion.
  test("midnight is local, not UTC", () => {
    const sent = query({ created_from: "2026-08-01" });
    const local = new Date(2026, 7, 1).toISOString();
    expect(sent.get("created_from")).toBe(local);
    expect(sent.get("created_from")).not.toBe(new Date("2026-08-01").toISOString());
  });

  test("a bare date never reaches the API", () => {
    const sent = query({ created_from: "2026-08-01", created_to: "2026-08-31" });
    expect(sent.get("created_from")).not.toBe("2026-08-01");
    expect(sent.get("created_to")).not.toBe("2026-08-31");
  });

  test("the other filters travel unchanged", () => {
    const sent = query({ status: "confirmed", stuck: "true", page: 2 });
    expect(sent.get("status")).toBe("confirmed");
    expect(sent.get("stuck")).toBe("true");
    expect(sent.get("page")).toBe("2");
  });

  test("page 1 and empty filters stay out of the request", () => {
    expect(adminOrderFiltersToQuery(emptyAdminOrderFilters)).toBe("");
  });
});
