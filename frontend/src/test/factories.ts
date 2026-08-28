import type { Listing, PublicProfile } from "../api/types";

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
    presence: { is_online: true },
    ...overrides,
  };
}
