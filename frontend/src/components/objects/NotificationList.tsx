import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useModal } from "../../providers/modalContext";
import type { Notification, NotificationKind } from "../../api/types";

// Where a kind sends the reader, declared once. The previous shape inferred it
// from which id happened to be set, with the else branch assuming a
// conversation - so a kind carrying neither an order nor a conversation opened
// the chat panel on `undefined`: no error, no empty state, just the wrong
// panel. Inference breaks the moment a third subject exists, and the schema
// now has four.
type Destination =
  | { via: "order" | "user" | "listing"; id: (n: Notification) => string | null }
  | { via: "conversation"; id: (n: Notification) => string | null };

const PATHS = { order: "orders", user: "users", listing: "listings" } as const;

const DESTINATIONS: Record<NotificationKind, Destination> = {
  order_placed: { via: "order", id: (n) => n.order_id },
  order_handed_over: { via: "order", id: (n) => n.order_id },
  order_cancelled: { via: "order", id: (n) => n.order_id },
  order_resolved: { via: "order", id: (n) => n.order_id },
  order_confirmed: { via: "order", id: (n) => n.order_id },
  order_completed: { via: "order", id: (n) => n.order_id },
  chat_request: { via: "conversation", id: (n) => n.conversation_id },
  review_received: { via: "user", id: (n) => n.actor_id },
  new_follower: { via: "user", id: (n) => n.actor_id },
  listing_removed: { via: "listing", id: (n) => n.listing_id },
  saved_listing_gone: { via: "listing", id: (n) => n.listing_id },
  saved_listing_deleted: { via: "user", id: (n) => n.actor_id },
};

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
        {/* An unknown kind falls back to the key itself in i18next, which is
            unreadable, so it gets a sentence of its own. A null title
            interpolates as the empty string rather than the word "null" - the
            kinds that have no listing do not mention one. */}
        {t([`notifications.kind.${notification.kind}`, "notifications.kind.unknown"], {
          title: notification.listing_title ?? "",
        })}
      </span>
    </>
  );
  const className = "hover:bg-surface-muted flex w-full gap-2 px-3 py-2 text-left";

  const destination = DESTINATIONS[notification.kind as NotificationKind] as
    Destination | undefined;
  const id = destination?.id(notification) ?? null;

  // A kind this build does not know, or one whose subject is missing, renders
  // as text and goes nowhere. Guessing a destination is what produced the
  // empty chat panel; showing nothing at all would hide a real event.
  if (!destination || id === null) {
    return <div className={className}>{inner}</div>;
  }

  if (destination.via !== "conversation") {
    return (
      <Link to={`/${PATHS[destination.via]}/${id}`} onClick={onNavigate} className={className}>
        {inner}
      </Link>
    );
  }

  // The chat panel is a modal, not a route, so this row cannot be a Link.
  return (
    <button
      type="button"
      onClick={() => {
        openChat(id);
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
