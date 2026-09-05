import { useState } from "react";
import { useTranslation } from "react-i18next";

import Button from "./Button";
import { isApiError } from "../../api/client";
import { useFollow, useFollowing, useUnfollow } from "../../api/follows";
import { useAuth } from "../../hooks/useAuth";
import { useModal } from "../../providers/modalContext";

export function FollowButton({ userId }: { userId: string }) {
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useTranslation();
  const { openModal } = useModal();
  const [error, setError] = useState<string | null>(null);

  const { data: following, isPending: followingPending } = useFollowing({
    enabled: Boolean(user),
  });
  const follow = useFollow();
  const unfollow = useUnfollow();

  if (user?.id === userId) return null;

  // AuthProvider reports user as null until restoreSession finishes, so acting
  // on !user here would offer a signed-in visitor the login modal.
  if (authLoading) {
    return (
      <Button variant="primary" disabled>
        {t("follows.follow")}
      </Button>
    );
  }

  if (!user) {
    return (
      <Button variant="primary" onClick={() => openModal("login")}>
        {t("follows.follow")}
      </Button>
    );
  }

  const isFollowing = following?.some((u) => u.id === userId) ?? false;
  const busy = follow.isPending || unfollow.isPending;

  async function toggle() {
    setError(null);
    try {
      await (isFollowing ? unfollow : follow).mutateAsync(userId);
    } catch (err) {
      setError(isApiError(err) ? err.message : t("follows.error"));
    }
  }

  return (
    <div className="space-y-2">
      <Button
        variant={isFollowing ? "secondary" : "primary"}
        disabled={busy || followingPending}
        onClick={() => void toggle()}
      >
        {busy ? t("follows.updating") : isFollowing ? t("follows.unfollow") : t("follows.follow")}
      </Button>
      {error && (
        <p role="alert" className="text-danger text-sm">
          {error}
        </p>
      )}
    </div>
  );
}
