import type {
  Conversation,
  ConversationListItem,
  Listing,
  Notification,
  Order,
  PublicProfile,
} from "../api/types";

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
    seller: {
      id: "3f1a7c2e-8b4d-4e91-9a5f-2c6d8e0b1a34",
      username: "mushroom_matti",
      avatar_url: null,
      // Enough reviews to be rated, so a test that wants the new-seller state
      // has to ask for it rather than getting it by default.
      rating: { average: 4.5, count: 12 },
    },
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

export function makeConversation(overrides: Partial<Conversation> = {}): Conversation {
  return {
    id: "0b8e3a25-6d4c-4f99-8a32-7c1e9b2f5d68",
    listing_id: "01a02305-b81c-7dcb-86a0-7f75e33e0af3",
    listing_title: "Golden Chanterelles",
    status: "accepted",
    role: "buyer",
    other_user: {
      id: SELLER_ID,
      username: "oscarroff",
      avatar_url: null,
      presence: { is_online: true },
    },
    created_at: "1970-01-01T00:00:00Z",
    updated_at: "1970-01-01T00:00:00Z",
    ...overrides,
  };
}

export function makeConversationListItem(
  overrides: Partial<ConversationListItem> = {},
): ConversationListItem {
  // Built off makeConversation so the two fixtures cannot drift. The list item
  // is the conversation minus created_at, plus the two list-only fields.
  // created_at is destructured off because the API does not send it on a list item
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { created_at: _created, ...base } = makeConversation();
  return {
    ...base,
    last_message: null,
    unread_count: 0,
    ...overrides,
  };
}

export function makeNotification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: "01a02305-b81c-7dcb-86a0-7f75e33e0af4",
    kind: "order_placed",
    listing_title: "Golden Chanterelles",
    order_id: "01a02305-b81c-7dcb-86a0-7f75e33e0af3",
    conversation_id: null,
    read_at: null,
    created_at: "1970-01-01T00:00:00Z",
    ...overrides,
  };
}
