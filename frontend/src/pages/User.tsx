// Public profile for any user, at /users/:id (#105).
//
// Reads GET /users/{id}, which returns less than /me/profile does: no email,
// no phone number, no date of birth. Those aren't hidden here, the backend
// never sends them - so don't try to render them.
import { useParams } from "react-router-dom";

import Avatar from "../components/objects/Avatar.tsx";
import Button from "../components/objects/Button.tsx";
import { ListingCard } from "../components/objects/ListingCard";
import { useModal } from "../providers/modalContext";
import { useAuth } from "../hooks/useAuth";
import { usePublicProfile } from "../api/profile";
import { useListings } from "../api/listings";
import { isApiError } from "../api/client";
import { deriveInitials } from "../lib/initials";
import NotFound from "./NotFound";

export default function User() {
  // string | undefined: TypeScript can't know this route has an ":id".
  const { id } = useParams();
  const { user } = useAuth();
  const { openChat } = useModal();

  const { data: profile, isLoading, error } = usePublicProfile(id);

  // TODO: there's no way to ask for one seller's listings, so we fetch them
  // all and filter here. Replace once the API supports it.
  const { data: allListings, isPending: listingsPending, isError: listingsError } = useListings();

  // 400 = the id isn't a UUID, 404 = no such user. Both mean "nothing here".
  if (isApiError(error) && (error.status === 404 || error.status === 400)) {
    return <NotFound />;
  }

  if (isLoading) return <p className="text-muted p-8 text-sm">Loading…</p>;

  if (error || !profile) {
    return (
      <p className="text-berry-500 p-8 text-sm">
        {isApiError(error) ? error.message : "Couldn't load this profile."}
      </p>
    );
  }

  // Your own page has no "message yourself" button.
  const isSelf = user?.id === profile.id;

  // profile.id, not the URL param: the API accepts an upper-case UUID and
  // answers with the canonical lower-case one, which is what listings carry.
  const listings = allListings?.filter((listing) => listing.seller_id === profile.id) ?? [];

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">User Profile</h1>

      <div className="flex flex-row gap-4">
        <div>
          <Avatar size="lg" initials={deriveInitials(profile.username)} />
        </div>
        <div className="text-accent my-auto flex flex-col text-base">
          <div className="font-bold">{profile.username}</div>
          <div className="text-muted flex items-center gap-2 text-sm">
            {/* A dot plus the word, so it doesn't rely on colour alone. */}
            <span
              aria-hidden="true"
              className={`h-2 w-2 rounded-full ${
                profile.presence.is_online ? "bg-accent" : "bg-surface-soft"
              }`}
            />
            {profile.presence.is_online ? "Online" : "Offline"}
          </div>
        </div>
      </div>

      {/* openChat() can't target a user yet - that arrives with the chat UI (#88). */}
      {!isSelf && (
        <Button variant="secondary" onClick={() => openChat()}>
          Message User
        </Button>
      )}

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Details</h2>
        <div className="grid max-w-fit grid-cols-2 gap-4">
          <div className="flex flex-col">
            <div className="text-muted">First Name</div>
            <div>{profile.firstname ?? "—"}</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Last Name</div>
            <div>{profile.lastname ?? "—"}</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Location</div>
            <div>{profile.location ?? "—"}</div>
          </div>
        </div>
      </div>

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Bio</h2>
        <div>{profile.bio ?? <span className="text-muted">No bio yet.</span>}</div>
      </div>

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Listings</h2>
        <p role="status" aria-live="polite" className="text-muted mt-4">
          {listingsPending && "Loading..."}
          {listingsError && "Couldn't load listings. Try again."}
          {!listingsPending && !listingsError && listings.length === 0 && "No listings yet!"}
        </p>
        {listings.length > 0 && (
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {listings.map((listing) => (
              <ListingCard key={listing.id} listing={listing} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
