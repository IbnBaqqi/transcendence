import { keys } from "../api/queryKeys";

// These look tautological, and that is the point: nothing else in the suite
// builds a key. Drop the id from users.detail and every profile shares one
// cache entry - open user B after user A and you see A's data under B's URL,
// with the whole suite still green.
describe("query keys", () => {
  test("two users get different detail keys", () => {
    expect(keys.users.detail("a")).not.toEqual(keys.users.detail("b"));
  });

  test("the same user gets a stable key", () => {
    expect(keys.users.detail("a")).toEqual(keys.users.detail("a"));
  });

  // Invalidation relies on prefix matching: keys.users.all must be a prefix of
  // every key beneath it, or invalidating the namespace silently does nothing.
  test("a detail key sits under the users namespace", () => {
    expect(keys.users.detail("a").slice(0, keys.users.all.length)).toEqual([...keys.users.all]);
  });

  test("listings and orders keep their ids too", () => {
    expect(keys.listings.detail("a")).not.toEqual(keys.listings.detail("b"));
    expect(keys.orders.detail("a")).not.toEqual(keys.orders.detail("b"));
  });

  // The admin user list is filtered and paged, so the query string has to be
  // part of the key. Without it, page 2 overwrites page 1 in the cache and
  // switching a filter shows the previous filter's rows.
  test("two filtered admin views get different list keys", () => {
    expect(keys.adminUsers.list("status=active")).not.toEqual(
      keys.adminUsers.list("status=suspended"),
    );
    expect(keys.adminUsers.list("page=1")).not.toEqual(keys.adminUsers.list("page=2"));
  });

  test("every admin user key sits under one namespace", () => {
    for (const key of [keys.adminUsers.list("status=active"), keys.adminUsers.history("a")]) {
      expect(key.slice(0, keys.adminUsers.all.length)).toEqual([...keys.adminUsers.all]);
    }
  });

  test("two accounts get different history keys", () => {
    expect(keys.adminUsers.history("a")).not.toEqual(keys.adminUsers.history("b"));
  });
});
