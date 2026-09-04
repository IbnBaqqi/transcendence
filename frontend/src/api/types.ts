// The API contract as the frontend sees it, mirroring backend/internal/dtos.
// Components import from here and never read raw response fields.
//
// No response envelope: endpoints return the payload directly and signal
// success with the status code. Errors are normalised into ApiError in
// client.ts.

// Timestamps arrive as ISO 8601 strings; parse where they're displayed.
export type Timestamp = string;

export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

// --- Listings ---

export interface ListingImage {
  id: string;
  url: string; // relative: "/uploads/abc.jpg"
  position: number;
}

export interface ListingSeller {
  id: string;
  username: string;
  avatar_url: string | null;
}

export interface Listing {
  id: string;
  seller_id: string;
  title: string;
  description: string;
  category: string;
  price: number;
  quantity: number;
  unit: string;
  created_at: Timestamp;
  updated_at: Timestamp;
  images: ListingImage[]; // always an array, never null
  seller: ListingSeller | null;
}

export interface Category {
  slug: string;
  name: string;
  children: Category[];
}

export type OrderStatus = "pending" | "confirmed" | "completed" | "cancelled" | "refunded";

export interface Order {
  id: string;
  listing_id: string;
  listing_title: string; // snapshotted at order time
  buyer_id: string;
  seller_id: string;
  quantity: number;
  // The backend sends these as strings ("18.00") while listing price is a
  // number. Normalised to number in orders.ts.
  unit_price: number;
  total_price: number;
  status: OrderStatus;
  // Null until that side confirms; both set means completed.
  seller_handed_over_at: Timestamp | null;
  buyer_received_at: Timestamp | null;
  created_at: Timestamp;
  updated_at: Timestamp;
}

// --- Chat ---

export interface AvatarResponse {
  avatar_url: string;
}

// GET /me/blocks. Deliberately not a ChatUser: that carries presence, and
// whether someone you blocked is online is exactly what stops being visible.
export interface BlockedUser {
  id: string;
  username: string;
  blocked_at: Timestamp | null;
}

export interface Presence {
  is_online: boolean;
  last_seen_at?: Timestamp; // absent when hidden AND when never seen
}

export interface ChatUser {
  id: string;
  username: string;
  avatar_url: string | null;
  presence: Presence;
}

export type ConversationStatus = "pending" | "accepted" | "declined";
export type ConversationRole = "buyer" | "seller";

export interface Conversation {
  id: string;
  listing_id: string | null; // null once the listing is deleted
  listing_title: string;
  status: ConversationStatus;
  role: ConversationRole;
  other_user: ChatUser;
  created_at: Timestamp;
  updated_at: Timestamp;
}

export interface MessagePreview {
  body: string;
  created_at: Timestamp;
}

export interface ConversationListItem extends Omit<Conversation, "created_at"> {
  last_message: MessagePreview | null;
  unread_count: number;
}

export interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  body: string;
  read_at?: Timestamp; // absent while unread
  created_at: Timestamp;
}

// --- Me ---

export interface UserSettings {
  show_online_status: boolean;
}

export interface UnreadCount {
  unread_count: number;
}

export type NotificationKind =
  "order_placed" | "order_handed_over" | "order_cancelled" | "order_resolved" | "chat_request";

export interface Notification {
  id: string;
  kind: NotificationKind;
  listing_title: string;
  order_id: string | null;
  conversation_id: string | null;
  read_at: Timestamp | null;
  created_at: Timestamp;
}

// Upper case, matching the backend's own values (auth/auth.go). A union rather
// than string so a typo or a lower-cased fixture fails the build instead of
// quietly failing an admin check.
export type UserRole = "USER" | "ADMIN";

// Auth Foundation additions. Mirrors backend UserInfo.
export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  // Whether the account can sign in with a password (false for a
  // provider-only account). Branch on this rather than providers being empty.
  has_password: boolean;
  // The OAuth providers linked to this account - empty for a password account.
  providers: string[];
}

export interface SignupInput {
  username: string;
  email: string;
  password: string;
}

// Login is by email only - a username is a display name, not a credential.
export interface LoginInput {
  email: string;
  password: string;
}

// POST /me/password. Requires a browser session (not an API key) and the
// current password. All other sessions are revoked; a fresh session is
// returned as a new refresh cookie.
export interface ChangePasswordInput {
  current_password: string;
  new_password: string;
}

// Returned by signup, login and refresh alike, so one code path can start a
// session from any of them. The refresh token itself is never here: it travels
// as an HttpOnly cookie.
export interface AuthResponse {
  access_token: string;
  user: User;
}

// GET /me/profile. Unset fields arrive as null rather than absent, so forms
// can render an empty input without deciding whether the key exists.
export interface OwnProfile {
  id: string;
  username: string;
  email: string;
  firstname: string | null;
  lastname: string | null;
  bio: string | null;
  phone_number: string | null;
  date_of_birth: string | null;
  location: string | null;
  // Always sent, null when no avatar is set - unlike presence below, the key
  // is never absent. The default avatar is the client's decision.
  avatar_url: string | null;
}

// PATCH /me/profile body. Each field has three states: key absent keeps the
// current value, "" or null clears it, a value replaces it (trimmed). Username
// and email are identity, not profile - they are not updatable here.
export type ProfileUpdateInput = Partial<{
  firstname: string | null;
  lastname: string | null;
  bio: string | null;
  phone_number: string | null;
  date_of_birth: string | null;
  location: string | null;
}>;

// GET /users/{id}. What everyone else sees.
//
// No email, phone_number or date_of_birth: the backend doesn't blank them
// out here, it never sends them at all (see PublicProfileResponse in
// backend/internal/dtos/profileDto.go). Don't add them to this type.
export interface PublicProfile {
  id: string;
  username: string;
  firstname: string | null;
  lastname: string | null;
  bio: string | null;
  location: string | null;
  avatar_url: string | null;
  // Absent for an anonymous caller: the API refuses to claim someone is
  // offline just because you are not signed in. Absent means "not shown",
  // which is a different fact from is_online: false.
  presence?: Presence;
}

// TODO: add an ApiError interface? e.g.
// export interface ApiError {
//   error: string;
//   details?: string;
// }

// NOTE: there is no response envelope. endpoints return the payload directly
// and signal success/failure through the HTTP status code. errors come back as
// { error, details } - the interceptor is where this will be normalised

// --- API keys ---

export interface ApiKey {
  id: string;
  name: string;
  // "fk_live_a3f9" - the only fragment of the key the server can return, since
  // it stores a SHA-256 hash of the rest.
  key_prefix: string;
  // Null until first use, and only written once a minute - "last seen", not a
  // request log. Neither field has omitempty, so both always arrive.
  last_used_at: Timestamp | null;
  revoked_at: Timestamp | null;
  created_at: Timestamp;
}

// Only ever returned by POST /me/api-keys. There is no endpoint that can show
// `key` again.
export interface CreatedApiKey extends ApiKey {
  key: string;
}

// --- Moderation (admin) ---

// Present tense: what an admin asks for.
export type ModerationRequestAction = "remove" | "restore" | "dismiss";

// Past tense: what the audit log records. Deliberately a different union from
// ModerationRequestAction - the two enums are not the same words, and sharing
// one type would let a request value into the log's renderer.
export type ModerationLogAction = "removed" | "restored" | "dismissed";

export type ReportReason = "spam" | "prohibited" | "misleading" | "offensive" | "other";

export type ReportStatus = "open" | "upheld" | "dismissed";

// One row of the queue: a listing with at least one open report. Grouped by
// listing, not by report - three complaints about one listing are one problem
// and one decision resolves all three.
export interface ReportedListing {
  listing_id: string;
  title: string;
  seller_id: string;
  removed_at: Timestamp | null;
  report_count: number;
  first_reported_at: Timestamp;
}

export interface Report {
  id: string;
  // Null once the reporter deletes their account: the complaint is about the
  // listing, not the person, so it outlives them.
  reporter_id: string | null;
  reason: ReportReason;
  // Attacker-controlled text. Render as plain text, never as markup.
  detail?: string;
  status: ReportStatus;
  created_at: Timestamp;
}

export interface ModerationAction {
  id: string;
  listing_id: string;
  // Null once that admin's account is deleted - an audit row that vanishes
  // with its author is not an audit row.
  moderator_id: string | null;
  action: ModerationLogAction;
  note?: string;
  created_at: Timestamp;
}

export interface ModerateListingInput {
  action: ModerationRequestAction;
  // Required by the API for "remove", optional otherwise.
  note?: string;
}

export interface ModerateListingResponse {
  listing: Listing;
  reports_resolved: number;
}

// --- Admin users ---

export type AdminUserStatus = "active" | "suspended" | "deleted";

// Past tense: the audit log records what was done. Distinct from the request
// bodies, which carry no action word at all - the endpoint is the verb.
export type UserActionKind = "suspended" | "reinstated" | "deleted";

// An account as an admin sees it. The only representation in the API carrying
// `email`, and the only one that shows deleted accounts - both justified by
// the route sitting behind RequireRole(ADMIN).
export interface AdminUser {
  id: string;
  // `deleted-<id>` once deleted: the row is anonymised rather than removed, so
  // nothing here identifies the person who left. Display "Deleted user".
  username: string;
  email: string;
  role: UserRole;
  // Derived at the response boundary, not stored. `deleted` wins over
  // `suspended`, because a deleted account is gone whatever it was before.
  status: AdminUserStatus;
  // Present while suspended, and still present on an account deleted while
  // suspended - why it was actioned is context an admin still wants.
  suspension_reason?: string;
  created_at: Timestamp;
  // Absent if never seen, and once deleted. Unlike every other view this
  // ignores show_online_status: it is a moderation signal, not presence.
  last_seen_at?: Timestamp;
}

export interface PaginatedAdminUsers {
  items: AdminUser[];
  total: number;
  // Echoes the requested page, even past the last one.
  page: number;
  limit: number;
  total_pages: number;
}

export interface UserAction {
  id: string;
  action: UserActionKind;
  // Empty only for a reinstatement, the one action needing no reason.
  note: string;
  // Null once that admin's own account is deleted - the record outlives them.
  moderator_id: string | null;
  created_at: Timestamp;
}

export interface SuspendUserInput {
  // Shown to the suspended user on their next request, so write it for them.
  reason: string;
}

export interface ReinstateUserInput {
  // Optional here alone: "this was a mistake" needs no justification the way a
  // punishment does.
  note?: string;
}

export interface DeleteUserInput {
  // The target's exact username, as confirmation. The same guard DELETE /me
  // uses - an admin deleting the wrong account cannot undo it.
  username: string;
  reason: string;
}

// --- Admin orders ---

// Order plus the one thing only an admin is told. Derived server-side and
// never stored, so it cannot drift from the columns and accounts it
// summarises - which is why the UI reads it rather than re-deriving it from
// the timestamps beside it.
export interface AdminOrder extends Order {
  // True when no party can move this order, which is exactly what /resolve
  // accepts. Two shapes qualify: confirmed with one handshake mark older than
  // seven days, or pending/confirmed with neither mark and both parties gone.
  stuck: boolean;
}

export interface PaginatedAdminOrders {
  items: AdminOrder[];
  total: number;
  // Echoes the requested page, even past the last one.
  page: number;
  limit: number;
  total_pages: number;
}

// completed says the trade happened after all; cancelled says it did not;
// refunded says it did and was undone. The two that end without a trade return
// the quantity to the listing's stock.
export type ResolveOutcome = "completed" | "cancelled" | "refunded";

export interface ResolveOrderInput {
  outcome: ResolveOutcome;
  // Required, max 500. It lands in the order's event history as the note and
  // both parties can read it, so it is written for them.
  reason: string;
}
