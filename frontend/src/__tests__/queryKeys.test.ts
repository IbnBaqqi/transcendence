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
});
