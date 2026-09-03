import { useTranslation } from "react-i18next";

import Avatar from "../objects/Avatar";
import { deriveThreadView } from "../../lib/chatState";
import { deriveInitials } from "../../lib/initials";
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
  const { other_user: other, last_message: last, unread_count: unread } = conversation;
  const view = deriveThreadView(conversation);

  return (
    <button
      type="button"
      onClick={onSelect}
      className="hover:bg-surface-muted flex w-full gap-3 p-3 text-left transition-colors"
    >
      <div className="relative shrink-0">
        <Avatar size="sm" initials={deriveInitials(other.username)} />
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

      <div className="min-w-0 flex-1">
        <div className="flex items-baseline justify-between gap-2">
          <span className="text-foreground truncate font-medium">{other.username}</span>
          {unread > 0 && (
            <span className="bg-accent text-accent-contrast shrink-0 rounded-full px-2 py-0.5 text-xs font-medium">
              {unread}
            </span>
          )}
        </div>

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
      </div>
    </button>
  );
}
