import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useModal } from "../../providers/modalContext";
import type { Notification } from "../../api/types";

function Row({
  notification,
  onNavigate,
}: {
  notification: Notification;
  onNavigate?: () => void;
}) {
  const { t } = useTranslation();
  const { openChat } = useModal();

  const unread = notification.read_at === null;
  const inner = (
    <>
      <span
        aria-hidden="true"
        className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${unread ? "bg-accent" : "bg-transparent"}`}
      />
      <span className={`text-sm ${unread ? "text-foreground font-medium" : "text-muted"}`}>
        {t(`notifications.kind.${notification.kind}`, { title: notification.listing_title })}
      </span>
    </>
  );
  const className = "hover:bg-surface-muted flex w-full gap-2 px-3 py-2 text-left";

  if (notification.order_id) {
    return (
      <Link to={`/orders/${notification.order_id}`} onClick={onNavigate} className={className}>
        {inner}
      </Link>
    );
  }

  // The chat panel is a modal, not a route, so this row cannot be a Link.
  return (
    <button
      type="button"
      onClick={() => {
        openChat(notification.conversation_id ?? undefined);
        onNavigate?.();
      }}
      className={className}
    >
      {inner}
    </button>
  );
}

export function NotificationList({
  notifications,
  onNavigate,
}: {
  notifications: Notification[];
  onNavigate?: () => void;
}) {
  return (
    <ul className="divide-line divide-y">
      {notifications.map((notification) => (
        <li key={notification.id}>
          <Row notification={notification} onNavigate={onNavigate} />
        </li>
      ))}
    </ul>
  );
}
