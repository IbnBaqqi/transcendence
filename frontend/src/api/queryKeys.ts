// Cache keys for TanStack Query, in one place.
//
// Two things use a key: the query that fills it and the invalidation that
// marks it stale after a write. If those two disagree by one character the
// invalidation silently does nothing and the UI shows stale data with no
// error - so both sides call these functions instead of writing arrays.
//
// Keys are matched by PREFIX, so invalidating keys.orders.all also
// invalidates every list and detail beneath it.

export const keys = {
  listings: {
    all: ["listings"] as const,
    detail: (id: string) => [...keys.listings.all, "detail", id] as const,
    // The query string is part of the key: different filters are different
    // cache entries, so going back to a previous search is instant.
    search: (query: string) => [...keys.listings.all, "search", query] as const,
    images: (id: string) => [...keys.listings.all, "images", id] as const,
  },

  orders: {
    all: ["orders"] as const,
    list: () => [...keys.orders.all, "list"] as const,
    detail: (id: string) => [...keys.orders.all, "detail", id] as const,
  },

  conversations: {
    all: ["conversations"] as const,
    list: () => [...keys.conversations.all, "list"] as const,
    detail: (id: string) => [...keys.conversations.all, "detail", id] as const,
    messages: (id: string) => [...keys.conversations.all, "messages", id] as const,
  },

  users: {
    all: ["users"] as const,
    // The id is part of the key, so each user is cached separately -
    // otherwise every profile would overwrite the last one.
    detail: (id: string) => [...keys.users.all, "detail", id] as const,
  },

  follows: {
    all: ["follows"] as const,
    // following() carries no user id, so it is only safe because AuthProvider
    // clears the whole cache on login and logout. Remove that clear, or write
    // to this key optimistically, and one account's follow list becomes the
    // next account's follow buttons.
    following: () => [...keys.follows.all, "following"] as const,
    followers: (id: string) => [...keys.follows.all, "followers", id] as const,
  },

  me: {
    all: ["me"] as const,
    profile: () => [...keys.me.all, "profile"] as const,
    settings: () => [...keys.me.all, "settings"] as const,
    unread: () => [...keys.me.all, "unread"] as const,
    blocks: () => [...keys.me.all, "blocks"] as const,
    notifications: () => [...keys.me.all, "notifications"] as const,
    saved: () => [...keys.me.all, "saved"] as const,
  },

  categories: {
    all: ["categories"] as const,
    list: () => [...keys.categories.all, "list"] as const,
  },
} as const;
