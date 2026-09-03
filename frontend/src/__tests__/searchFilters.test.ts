import {
  activeFilters,
  emptyFilters,
  filtersToParams,
  parseFilters,
  toSearchQuery,
  withFilters,
  PAGE_SIZE,
} from "../lib/searchFilters";

describe("withFilters", () => {
  // The bug this exists for: filter down to three results while sitting on
  // page 5 and you get a valid, empty page.
  test("changing a filter sends you back to page 1", () => {
    const onPage5 = { ...emptyFilters, page: 5 };
    expect(withFilters(onPage5, { category: "mushrooms" }).page).toBe(1);
  });

  test("but paging keeps the page it was given", () => {
    const filtered = { ...emptyFilters, category: "mushrooms" };
    const next = withFilters(filtered, { page: 2 });
    expect(next.page).toBe(2);
    expect(next.category).toBe("mushrooms");
  });
});

describe("parseFilters", () => {
  test("reads the filters out of the query string", () => {
    const f = parseFilters(
      new URLSearchParams("keyword=chanterelle&category=mushrooms&sort=price_asc&page=3"),
    );
    expect(f).toEqual({
      ...emptyFilters,
      keyword: "chanterelle",
      category: "mushrooms",
      sort: "price_asc",
      page: 3,
    });
  });

  // A URL is user input: it can be typed, edited, or come from a stale link.
  test.each(["page=0", "page=-2", "page=abc", "page=1.5", ""])(
    "an unusable page (%s) falls back to 1",
    (query) => {
      expect(parseFilters(new URLSearchParams(query)).page).toBe(1);
    },
  );

  test("a sort the backend would reject falls back to newest", () => {
    expect(parseFilters(new URLSearchParams("sort=cheapest")).sort).toBe("newest");
  });

  test("a non-numeric price is dropped rather than forwarded as NaN", () => {
    const f = parseFilters(new URLSearchParams("min_price=abc&max_price=-5"));
    expect(f.min_price).toBe("");
    expect(f.max_price).toBe("");
  });
});

describe("filtersToParams", () => {
  test("an untouched search has no query string at all", () => {
    expect(filtersToParams(emptyFilters).toString()).toBe("");
  });

  test("round-trips through the URL", () => {
    const filters = {
      ...emptyFilters,
      keyword: "birch bolete",
      category: "mushrooms",
      min_price: "5",
      sort: "price_desc" as const,
      page: 2,
    };
    expect(parseFilters(filtersToParams(filters))).toEqual(filters);
  });
});

describe("toSearchQuery", () => {
  test("prices reach the API as numbers, and absent ones not at all", () => {
    const q = toSearchQuery({ ...emptyFilters, min_price: "5.50" });
    expect(q.min_price).toBe(5.5);
    expect(q.max_price).toBeUndefined();
    expect(q.limit).toBe(PAGE_SIZE);
  });
});

describe("activeFilters", () => {
  test("lists only the filters that are set", () => {
    const f = { ...emptyFilters, category: "berries", max_price: "12", page: 4 };
    expect(activeFilters(f)).toEqual([
      { key: "category", value: "berries" },
      { key: "max_price", value: "12" },
    ]);
  });
});
