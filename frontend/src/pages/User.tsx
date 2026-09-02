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
import { useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { isApiError } from "../api/client";
import { deriveInitials } from "../lib/initials";
import NotFound from "./NotFound";
import { useTranslation } from "react-i18next";

export default function User() {
  const { t } = useTranslation();
  // string | undefined: TypeScript can't know this route has an ":id".
  const { id } = useParams();
  const { user } = useAuth();
  const { openChat } = useModal();

  const { data: profile, isLoading, error } = usePublicProfile(id);

  // The URL param rather than profile.id, so the query can start before the
  // profile resolves: the backend compares uuids, so the casing the API
  // canonicalises does not matter to it the way it did to a string filter here.
  const {
    data: sellerListings,
    isPending: listingsPending,
    isError: listingsError,
  } = useSearchListings({ seller_id: id, limit: 20 });
  const categoryName = useLocalizedCategoryNames();

  // 400 = the id isn't a UUID, 404 = no such user. Both mean "nothing here".
  if (isApiError(error) && (error.status === 404 || error.status === 400)) {
    return <NotFound />;
  }

  if (isLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (error || !profile) {
    return (
      <p className="text-berry-500 p-8 text-sm">
        {isApiError(error) ? error.message : t("pages.user.profileError")}
      </p>
    );
  }

  // Your own page has no "message yourself" button.
  const isSelf = user?.id === profile.id;

  const listings = sellerListings?.items ?? [];
  const hiddenListings = (sellerListings?.total ?? 0) - listings.length;

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.user.title")}</h1>

      <div className="flex flex-row gap-4">
        <div>
          {/* null when no avatar is set, and Avatar falls back to initials.
              ?? undefined because the prop is optional, not nullable. */}
          <Avatar
            size="lg"
            initials={deriveInitials(profile.username)}
            imageUrl={profile.avatar_url ?? undefined}
          />
        </div>
        <div className="text-accent my-auto flex flex-col text-base">
          <div className="font-bold">{profile.username}</div>
          {/* No presence field means we are not signed in, not that they are
              offline - so show nothing rather than assert a falsehood. */}
          {profile.presence && (
            <div className="text-muted flex items-center gap-2 text-sm">
              {/* A dot plus the word, so it doesn't rely on colour alone. */}
              <span
                aria-hidden="true"
                className={`h-2 w-2 rounded-full ${
                  profile.presence.is_online ? "bg-accent" : "bg-surface-soft"
                }`}
              />
              {profile.presence.is_online ? t("pages.user.online") : t("pages.user.offline")}
            </div>
          )}
        </div>
      </div>

      {/* openChat() can't target a user yet - that arrives with the chat UI (#88). */}
      {!isSelf && (
        <Button variant="secondary" onClick={() => openChat()}>
          {t("pages.user.messageUser")}
        </Button>
      )}

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">{t("pages.user.details")}</h2>
        <div className="grid max-w-fit grid-cols-2 gap-4">
          <div className="flex flex-col">
            <div className="text-muted">{t("pages.user.firstName")}</div>
            <div>{profile.firstname ?? "—"}</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">{t("pages.user.lastName")}</div>
            <div>{profile.lastname ?? "—"}</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">{t("pages.user.location")}</div>
            <div>{profile.location ?? "—"}</div>
          </div>
        </div>
      </div>

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">{t("pages.user.bio")}</h2>
        <div>{profile.bio ?? <span className="text-muted">{t("pages.user.noBio")}</span>}</div>
      </div>

      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">{t("pages.user.listings")}</h2>
        <p role="status" aria-live="polite" className="text-muted mt-4">
          {listingsPending && t("pages.user.loadingListings")}
          {listingsError && t("pages.user.listingsError")}
          {!listingsPending &&
            !listingsError &&
            listings.length === 0 &&
            t("pages.user.noListings")}
        </p>
        {hiddenListings > 0 && (
          <p className="text-muted mt-2 text-sm">
            Showing {listings.length} of {sellerListings!.total}
          </p>
        )}
        {listings.length > 0 && (
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {listings.map((listing) => (
              <ListingCard
                key={listing.id}
                listing={listing}
                categoryName={categoryName(listing.category)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
