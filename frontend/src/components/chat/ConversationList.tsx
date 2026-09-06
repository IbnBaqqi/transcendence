import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import Avatar from "../objects/Avatar";
import { deriveThreadView } from "../../lib/chatState";
import { deriveInitials } from "../../lib/initials";
import { useModal } from "../../providers/modalContext";
import type { ConversationListItem } from "../../api/types";

export function ConversationList({
  conversations,
  onSelect,
}: {
  conversations: ConversationListItem[];
  onSelect: (id: string) => void;
}) {
  const { t } = useTranslation();

  if (conversations.length === 0) {
    return <p className="text-muted p-4 text-sm">{t("chat.empty")}</p>;
  }

  return (
    <ul className="divide-line divide-y">
      {conversations.map((conversation) => (
        <li key={conversation.id}>
          <ConversationRow conversation={conversation} onSelect={() => onSelect(conversation.id)} />
        </li>
      ))}
    </ul>
  );
}

function ConversationRow({
  conversation,
  onSelect,
}: {
  conversation: ConversationListItem;
  onSelect: () => void;
}) {
  const { t } = useTranslation();
  const { closeChat } = useModal();
  const { other_user: other, last_message: last, unread_count: unread } = conversation;
  const view = deriveThreadView(conversation);

  // Closing the chat first: it is a floating modal, so without that the user
  // page would render underneath an open panel that stays on screen.
  const openProfile = () => closeChat();

  return (
    <div className="flex items-center gap-3 p-3">
      <Link
        to={`/users/${other.id}`}
        onClick={openProfile}
        aria-label={t("chat.viewProfile", { username: other.username })}
        title={t("chat.viewProfile", { username: other.username })}
        className="focus:ring-line shrink-0 rounded-full focus:ring-2 focus:ring-offset-2 focus:outline-none"
      >
        <div className="relative">
          <Avatar
            size="sm"
            initials={deriveInitials(other.username)}
            imageUrl={other.avatar_url ?? undefined}
            interactive
          />
          {/* Colour-only, unlike the profile page which pairs the dot with the
              word: there is no room in a row this narrow. Tracked with the other
              a11y items in #27. */}
          {other.presence.is_online && (
            <span
              aria-hidden="true"
              className="bg-accent border-surface absolute right-0 bottom-0 h-2.5 w-2.5 rounded-full border-2"
            />
          )}
        </div>
      </Link>

      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <div className="flex items-baseline justify-between gap-2">
          <Link
            to={`/users/${other.id}`}
            onClick={openProfile}
            aria-label={t("chat.viewProfile", { username: other.username })}
            title={t("chat.viewProfile", { username: other.username })}
            className="text-foreground focus:ring-accent min-w-0 truncate font-medium hover:underline focus:ring-2 focus:ring-offset-2 focus:outline-none"
          >
            {other.username}
          </Link>
          {unread > 0 && (
            <span className="bg-accent text-accent-contrast shrink-0 rounded-full px-2 py-0.5 text-xs font-medium">
              {unread}
            </span>
          )}
        </div>

        <button
          type="button"
          onClick={onSelect}
          aria-label={t("chat.openThread", { username: other.username })}
          className="hover:bg-surface-soft w-full rounded-md px-1 py-1 text-left transition-colors"
        >
          <p className="text-muted truncate text-sm">{conversation.listing_title}</p>

          {/* A pending or declined thread has no useful preview - what matters
              is that it is waiting on someone. */}
          {conversation.status === "accepted" ? (
            <p className="text-muted truncate text-sm">
              {last ? last.body : <span className="italic">{t("chat.noMessagePreview")}</span>}
            </p>
          ) : (
            <p className="text-muted truncate text-sm italic">{t(view.statusKey)}</p>
          )}
        </button>
      </div>
    </div>
  );
}
