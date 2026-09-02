import type { Listing, Order, PublicProfile } from "../api/types";

// Override only what a test cares about, so adding a field to Listing doesn't
// mean editing every test that builds one.
export function makeListing(overrides: Partial<Listing> = {}): Listing {
  return {
    id: "01a02305-b81c-7dcb-86a0-7f75e33e0af3",
    seller_id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    title: "Golden Chanterelles",
    description: "Freshly foraged this morning.",
    category: "mushrooms",
    price: 18,
    quantity: 4,
    unit: "kg",
    created_at: "1970-01-01T00:00:00Z",
    updated_at: "1970-01-01T00:00:00Z",
    images: [],
    ...overrides,
  };
}

// The default id matches makeListing's seller_id, so a listing built with no
// overrides already belongs to this user.
export function makePublicProfile(overrides: Partial<PublicProfile> = {}): PublicProfile {
  return {
    id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
    username: "oscarroff",
    firstname: "Oscar",
    lastname: "Rogers",
    bio: "Forages the Nuuksio area.",
    location: "Helsinki",
    avatar_url: null,
    presence: { is_online: true },
    ...overrides,
  };
}

// Shared between the order factory and the tests that assert on roles.
// SELLER_ID matches makeListing's seller_id, so a default order and a default
// listing belong to the same seller.
export const SELLER_ID = "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34";
export const BUYER_ID = "9c4e1b7a-2d63-4f80-8e15-77b2a4c9d0e6";

export function makeOrder(overrides: Partial<Order> = {}): Order {
  return {
    id: "0a7d2f14-5c3b-4e88-9f21-6b0d8a1e4c57",
    listing_id: "01a02305-b81c-7dcb-86a0-7f75e33e0af3",
    listing_title: "Golden Chanterelles",
    buyer_id: BUYER_ID,
    seller_id: SELLER_ID,
    quantity: 2,
    unit_price: 18,
    total_price: 36,
    status: "pending",
    seller_handed_over_at: null,
    buyer_received_at: null,
    created_at: "1970-01-01T00:00:00Z",
    updated_at: "1970-01-01T00:00:00Z",
    ...overrides,
  };
}
