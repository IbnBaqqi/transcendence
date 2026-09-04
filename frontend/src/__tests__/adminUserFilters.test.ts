import {
  adminUserFiltersToParams,
  emptyAdminUserFilters,
  parseAdminUserFilters,
  withAdminUserFilters,
} from "../lib/adminUserFilters";

const parse = (query: string) => parseAdminUserFilters(new URLSearchParams(query));

describe("parsing", () => {
  test("reads the filters it recognises", () => {
    expect(parse("role=ADMIN&status=suspended&page=3")).toEqual({
      role: "ADMIN",
      status: "suspended",
      page: 3,
    });
  });

  test("an empty query is the default view", () => {
    expect(parse("")).toEqual(emptyAdminUserFilters);
  });

  // A URL is user input. The API answers an unknown value with a 400, which
  // reads to the reader as "the server is down" rather than "you typed that".
  test.each([
    ["role=banana", "role"],
    ["role=admin", "role"],
    ["status=banned", "status"],
    ["status=Active", "status"],
  ] as const)("%s is dropped rather than forwarded", (query, field) => {
    expect(parse(query)[field]).toBe("");
  });

  test.each(["page=abc", "page=0", "page=-2", "page=2.5", "page=1e30", "page=9999999999"])(
    "%s falls back to page 1",
    (query) => {
      expect(parse(query).page).toBe(1);
    },
  );
});

describe("updating", () => {
  // Filtering down to three accounts while on page 5 would otherwise strand
  // the reader on a page that is valid and empty.
  test("changing a filter returns to page 1", () => {
    const onPageFive = { role: "USER", status: "", page: 5 } as const;
    expect(withAdminUserFilters(onPageFive, { status: "suspended" }).page).toBe(1);
  });

  test("changing the page keeps the page", () => {
    const filters = { role: "USER", status: "", page: 1 } as const;
    expect(withAdminUserFilters(filters, { page: 4 }).page).toBe(4);
  });

  test("clearing a filter also returns to page 1", () => {
    const filtered = { role: "ADMIN", status: "", page: 6 } as const;
    expect(withAdminUserFilters(filtered, { role: "" })).toEqual({
      role: "",
      status: "",
      page: 1,
    });
  });
});

describe("serialising", () => {
  test("an untouched list has no query string at all", () => {
    expect(adminUserFiltersToParams(emptyAdminUserFilters).toString()).toBe("");
  });

  test("page 1 stays out of the URL", () => {
    expect(adminUserFiltersToParams({ role: "ADMIN", status: "", page: 1 }).toString()).toBe(
      "role=ADMIN",
    );
  });

  test("what goes in comes back out", () => {
    const filters = { role: "ADMIN", status: "deleted", page: 7 } as const;
    expect(parseAdminUserFilters(adminUserFiltersToParams(filters))).toEqual(filters);
  });
});
