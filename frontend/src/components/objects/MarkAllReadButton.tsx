import { useTranslation } from "react-i18next";

import { useMarkNotificationsRead } from "../../api/notifications";

export function MarkAllReadButton({ unread, className }: { unread: number; className?: string }) {
  const { t } = useTranslation();
  const markRead = useMarkNotificationsRead();

  if (unread === 0) return null;

  return (
    <>
      <button
        type="button"
        onClick={() => markRead.mutate()}
        disabled={markRead.isPending}
        className={`text-accent cursor-pointer disabled:opacity-50 ${className ?? ""}`}
      >
        {t("notifications.markAllRead")}
      </button>
      {markRead.isError && (
        <p className="text-muted text-xs">{t("notifications.markAllReadError")}</p>
      )}
    </>
  );
}
