import { useTranslation } from "react-i18next";

import { unreadCount, useNotifications } from "../api/notifications";
import { isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useModal } from "../providers/modalContext";
import Button from "../components/objects/Button";
import { MarkAllReadButton } from "../components/objects/MarkAllReadButton";
import { NotificationList } from "../components/objects/NotificationList";
import { Skeleton } from "../components/objects/Skeleton";

export default function Notifications() {
  const { t } = useTranslation();
  const { user, isLoading: authLoading } = useAuth();
  const { openModal } = useModal();

  const {
    data: notifications,
    isPending,
    isError,
    error,
    refetch,
  } = useNotifications({ enabled: Boolean(user) });

  if (authLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (!user) {
    return (
      <div className="mx-auto max-w-3xl space-y-3 px-4 py-8">
        <h1 className="text-foreground text-page-title font-bold">
          {t("pages.notifications.title")}
        </h1>
        <p className="text-muted text-sm">{t("pages.notifications.signedOut")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-foreground text-page-title font-bold">
          {t("pages.notifications.title")}
        </h1>
        <MarkAllReadButton unread={unreadCount(notifications)} className="text-sm" />
      </div>

      {isPending && (
        <div className="mt-6 space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      )}

      {isError && !notifications && (
        <Skeleton
          variant="error"
          className="mt-6 h-24 w-full"
          message={isApiError(error) ? error.message : t("pages.notifications.error")}
          onRetry={() => refetch()}
        />
      )}

      {notifications?.length === 0 && (
        <div className="border-line mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">{t("notifications.empty")}</p>
        </div>
      )}

      {notifications && notifications.length > 0 && (
        <div className="border-line bg-surface mt-6 rounded-lg border">
          <NotificationList notifications={notifications} />
        </div>
      )}
    </div>
  );
}
