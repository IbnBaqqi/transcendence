import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { unreadCount, useNotifications } from "../../api/notifications";
import { useAuth } from "../../hooks/useAuth";
import { MarkAllReadButton } from "./MarkAllReadButton";
import { NotificationList } from "./NotificationList";

export function NotificationBell() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const signedIn = Boolean(user);

  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);

  const { data: notifications, isError } = useNotifications({ enabled: signedIn });
  const unread = unreadCount(notifications);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (rootRef.current && !rootRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  if (!signedIn) return null;

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t("nav.notifications")}
        aria-expanded={open}
        className="text-muted hover:text-foreground relative cursor-pointer"
      >
        <svg className="h-6 w-5" aria-hidden="true">
          <use href="/icons.svg#notifications-icon" />
        </svg>
        {unread > 0 && (
          <span
            aria-label={t("notifications.unread", { count: unread })}
            className="bg-accent text-accent-contrast absolute -top-1 -right-1.5 min-w-4 rounded-full px-1 text-[10px] leading-4 font-medium"
          >
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </button>

      {open && (
        <div
          role="group"
          aria-label={t("nav.notifications")}
          className="border-line bg-surface absolute top-full right-0 z-20 mt-2 w-80 rounded border shadow-lg"
        >
          <div className="border-line flex items-center justify-between border-b px-3 py-2">
            <h2 className="text-foreground text-sm font-bold">{t("nav.notifications")}</h2>
            <MarkAllReadButton unread={unread} className="text-xs" />
          </div>

          {!notifications ? (
            // Same branch, two meanings: with no data the panel is either
            // still waiting or has given up. "See all" below is the way out -
            // the page has the retry.
            <p className="text-muted px-3 py-4 text-sm">
              {isError ? t("pages.notifications.error") : t("common.loading")}
            </p>
          ) : notifications.length === 0 ? (
            <p className="text-muted px-3 py-4 text-sm">{t("notifications.empty")}</p>
          ) : (
            <NotificationList notifications={notifications} onNavigate={() => setOpen(false)} />
          )}

          <Link
            to="/notifications"
            onClick={() => setOpen(false)}
            className="border-line text-accent hover:bg-surface-muted block border-t px-3 py-2 text-center text-xs"
          >
            {t("notifications.seeAll")}
          </Link>
        </div>
      )}
    </div>
  );
}
