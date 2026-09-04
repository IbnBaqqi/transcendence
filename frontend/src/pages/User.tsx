// Public profile for any user, at /users/:id (#105).
//
// Reads GET /users/{id}, which returns less than /me/profile does: no email,
// no phone number, no date of birth. Those aren't hidden here, the backend
// never sends them - so don't try to render them.
import { useParams } from "react-router-dom";

import Avatar from "../components/objects/Avatar.tsx";
import Button from "../components/objects/Button.tsx";
import { ListingCard } from "../components/objects/ListingCard";
import { BlockButton } from "../components/objects/BlockButton";
import { FollowButton } from "../components/objects/FollowButton";
import { PresenceIndicator } from "../components/objects/PresenceIndicator";
import { Pagination } from "../components/objects/Pagination";
import { useModal } from "../providers/modalContext";
import { useAuth } from "../hooks/useAuth";
import { usePublicProfile } from "../api/profile";
import { useFollowers } from "../api/follows";
import { useBlocks } from "../api/blocks";
import { useSearchListings } from "../api/listings";
import { useLocalizedCategoryNames } from "../api/categories";
import { usePageParam } from "../hooks/usePageParam";
import { PAGE_SIZE } from "../lib/searchFilters";
import { isApiError } from "../api/client";
import { deriveInitials } from "../lib/initials";
import NotFound from "./NotFound";
import { useTranslation } from "react-i18next";

export default function User() {
  const { t } = useTranslation();
  // string | undefined: TypeScript can't know this route has an ":id".
  const { id } = useParams();
  const { user, isLoading: authLoading } = useAuth();
  const { openChat } = useModal();

  const { data: profile, isLoading, error } = usePublicProfile(id);

  // The URL param rather than profile.id, so the query can start before the
  // profile resolves: the backend compares uuids, so the casing the API
  // canonicalises does not matter to it the way it did to a string filter here.
  const [page, setPage] = usePageParam();
  const {
    data: sellerListings,
    isPending: listingsPending,
    isError: listingsError,
  } = useSearchListings({ seller_id: id, page, limit: PAGE_SIZE });
  const categoryName = useLocalizedCategoryNames();

  // profile.id, not the route param: query keys are compared as strings, and
  // the follow button invalidates the key built from profile.id. A URL with
  // different casing would leave this count one behind, with nothing to show
  // that the invalidation missed.
  const { data: followers } = useFollowers(profile?.id, user?.id);
  const signedIn = Boolean(user);
  const { data: blocks, isPending: blocksPending } = useBlocks({ enabled: signedIn });

  // 400 = the id isn't a UUID, 404 = no such user. Both mean "nothing here".
  if (isApiError(error) && (error.status === 404 || error.status === 400)) {
    return <NotFound />;
  }

  // authLoading too: isSelf below is false while the viewer is unknown, so
  // your own profile offers you a button to message yourself.
  if (authLoading || isLoading)
    return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (error || !profile) {
    return (
      <p className="text-berry-500 p-8 text-sm">
        {isApiError(error) ? error.message : t("pages.user.profileError")}
      </p>
    );
  }

  // Your own page has no "message yourself" button.
  const isSelf = user?.id === profile.id;
  // Unknown until the list arrives, and false would mean "not blocked": the
  // message button would appear on a blocked profile and then be taken away.
  // signedIn first, because a disabled query reports pending forever - without
  // it every logged-out visitor is told they blocked this person.
  const isBlocked =
    signedIn && (blocksPending || (blocks?.some((b) => b.id === profile.id) ?? false));

  const listings = sellerListings?.items ?? [];

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.user.title")}</h1>

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
          {profile.presence && <PresenceIndicator presence={profile.presence} />}
          {followers && (
            <div className="text-muted text-sm">
              {t("follows.followers", { count: followers.length })}
            </div>
          )}
        </div>
      </div>

      <div className="flex flex-wrap items-start gap-3">
        {/* openChat() can't target a user yet - that arrives with #88 - so this
            opens the inbox. Hidden once blocked because sending would 403.
            Following is not hidden: the backend has no block check on it, and
            the friends list keeps offering it for the same person. */}
        {!isSelf && !isBlocked && (
          <Button variant="secondary" onClick={() => openChat()}>
            {t("pages.user.messageUser")}
          </Button>
        )}
        <FollowButton userId={profile.id} />
        <BlockButton userId={profile.id} />
      </div>

      {isBlocked && <p className="text-muted text-sm">{t("blocks.blocked")}</p>}

      <div className="space-y-1">
        <h2 className="text-foreground text-section font-bold">{t("pages.user.details")}</h2>
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
        <h2 className="text-foreground text-section font-bold">{t("pages.user.bio")}</h2>
        <div>{profile.bio ?? <span className="text-muted">{t("pages.user.noBio")}</span>}</div>
      </div>

      <div className="space-y-1">
        <h2 className="text-foreground text-section font-bold">{t("pages.user.listings")}</h2>
        <p role="status" aria-live="polite" className="text-muted mt-4">
          {listingsPending && t("pages.user.loadingListings")}
          {listingsError && t("pages.user.listingsError")}
          {!listingsPending &&
            !listingsError &&
            sellerListings?.total === 0 &&
            t("pages.user.noListings")}
        </p>
        {listings.length > 0 && (
          <div className="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-3">
            {listings.map((listing) => (
              <ListingCard
                key={listing.id}
                listing={listing}
                categoryName={categoryName(listing.category)}
                showSeller={false}
              />
            ))}
          </div>
        )}
        {sellerListings && sellerListings.total > 0 && (
          <div className="mt-6">
            <Pagination
              page={sellerListings.page}
              totalPages={sellerListings.total_pages}
              total={sellerListings.total}
              onPageChange={setPage}
            />
          </div>
        )}
      </div>
    </div>
  );
}
