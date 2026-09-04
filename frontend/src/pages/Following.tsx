import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useFollowing } from "../api/follows";
import { isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useModal } from "../providers/modalContext";
import Avatar from "../components/objects/Avatar";
import Button from "../components/objects/Button";
import { FollowButton } from "../components/objects/FollowButton";
import { PresenceIndicator } from "../components/objects/PresenceIndicator";
import { Skeleton } from "../components/objects/Skeleton";
import { deriveInitials } from "../lib/initials";

export default function Following() {
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useTranslation();
  const { openModal } = useModal();

  const {
    data: following,
    isPending,
    isError,
    error,
    refetch,
  } = useFollowing({ enabled: Boolean(user) });

  if (authLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (!user) {
    return (
      <div className="mx-auto max-w-2xl space-y-3 px-4 py-8">
        <h1 className="text-foreground text-page-title font-bold">{t("pages.following.title")}</h1>
        <p className="text-muted text-sm">{t("pages.following.signedOut")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.following.title")}</h1>

      {isPending && (
        <div className="mt-6 space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      )}

      {isError && !following && (
        <Skeleton
          variant="error"
          className="mt-6 h-24 w-full"
          message={isApiError(error) ? error.message : t("pages.following.error")}
          onRetry={() => refetch()}
        />
      )}

      {following?.length === 0 && (
        <div className="border-line mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">{t("pages.following.empty")}</p>
          <p className="text-muted mt-1 text-sm">{t("pages.following.emptyHint")}</p>
        </div>
      )}

      {following && following.length > 0 && (
        <ul className="mt-6 space-y-3">
          {following.map((followee) => (
            <li
              key={followee.id}
              className="border-line bg-surface flex items-center gap-3 rounded-lg border p-3"
            >
              <Avatar
                size="sm"
                initials={deriveInitials(followee.username)}
                imageUrl={followee.avatar_url ?? undefined}
              />
              <div className="min-w-0 flex-1">
                <Link
                  to={`/users/${followee.id}`}
                  className="text-foreground font-medium hover:underline"
                >
                  {followee.username}
                </Link>
                <PresenceIndicator presence={followee.presence} />
              </div>
              <FollowButton userId={followee.id} />
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
