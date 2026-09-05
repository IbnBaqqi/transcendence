import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { unreadCount, useNotifications } from "../../api/notifications";
import { useAuth } from "../../hooks/useAuth";
import { useDismissable } from "../../hooks/useDismissable";
import { MarkAllReadButton } from "./MarkAllReadButton";
import { NotificationList } from "./NotificationList";

export function NotificationBell() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const signedIn = Boolean(user);

  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  useDismissable(open, setOpen, rootRef, triggerRef);

  const { data: notifications, isError } = useNotifications({ enabled: signedIn });
  const unread = unreadCount(notifications);

  if (!signedIn) return null;

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t("nav.notifications")}
        aria-expanded={open}
        className="text-muted hover:text-foreground relative inline-flex cursor-pointer items-center"
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
          // Anchored to the bell from sm up, but the bell sits ~100px from the
          // right edge, so a 320px panel hung off it starts off-screen on a
          // phone. Below sm it leaves the anchor and spans the viewport.
          className="border-line bg-surface fixed inset-x-2 top-14 z-20 rounded border shadow-lg sm:absolute sm:inset-x-auto sm:top-full sm:right-0 sm:mt-2 sm:w-80"
        >
          <div className="border-line flex items-center justify-between border-b px-3 py-2">
            <h2 className="text-foreground text-secondary font-bold">{t("nav.notifications")}</h2>
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
