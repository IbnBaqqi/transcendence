import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

import {
  useAcceptConversation,
  useConversation,
  useDeclineConversation,
  useMarkConversationRead,
  useMessages,
  useSendMessage,
} from "../../api/conversations";
import { isApiError } from "../../api/client";
import { deriveThreadView } from "../../lib/chatState";
import { useAuth } from "../../hooks/useAuth";
import Button from "../objects/Button";
import { Skeleton } from "../objects/Skeleton";
import type { Message } from "../../api/types";

export function MessageThread({
  conversationId,
  onBack,
}: {
  conversationId: string;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { data: conversation, isPending, isError, refetch } = useConversation(conversationId);
  const { data: messages } = useMessages(conversationId);

  const send = useSendMessage(conversationId);
  const accept = useAcceptConversation();
  const decline = useDeclineConversation();
  const markRead = useMarkConversationRead();

  const [draft, setDraft] = useState("");
  const [error, setError] = useState<string | null>(null);

  // Opening the thread clears its badge. Guarded by a ref because React runs
  // effects twice in dev, and because the mutation object is new every render -
  // the ref, not the dependency list, is what stops this looping.
  const markedRef = useRef<string | null>(null);
  useEffect(() => {
    if (conversationId && markedRef.current !== conversationId) {
      markedRef.current = conversationId;
      markRead.mutate(conversationId);
    }
  }, [conversationId, markRead]);

  if (isPending) return <Skeleton className="m-4 h-40" />;

  if (isError || !conversation) {
    return (
      <Skeleton
        variant="error"
        className="m-4 h-40"
        message={t("chat.threadError")}
        onRetry={() => refetch()}
      />
    );
  }

  const view = deriveThreadView(conversation);
  const other = conversation.other_user;

  async function handleSend(event: React.FormEvent) {
    event.preventDefault();
    const body = draft.trim();
    if (!body) return;

    setError(null);
    try {
      await send.mutateAsync(body);
      setDraft("");
    } catch (err) {
      // 409 = the thread isn't accepted (or was declined while this sat open);
      // 403 = blocked, which the conversation itself never tells us about.
      setError(isApiError(err) ? err.message : t("chat.sendFailed"));
    }
  }

  async function decide(decision: "accept" | "decline") {
    setError(null);
    try {
      await (decision === "accept" ? accept : decline).mutateAsync(conversationId);
    } catch (err) {
      setError(isApiError(err) ? err.message : t("chat.decideFailed"));
    }
  }

  return (
    <div className="flex h-full flex-col">
      <div className="border-line flex items-center gap-3 border-b px-4 py-3">
        <Button variant="tertiary" onClick={onBack}>
          {t("chat.back")}
        </Button>
        <div className="min-w-0">
          <p className="text-foreground truncate font-medium">
            {other.username}
            {other.presence.is_online && (
              <span className="text-muted text-sm"> · {t("chat.online")}</span>
            )}
          </p>
          <p className="text-muted truncate text-sm">{conversation.listing_title}</p>
        </div>
      </div>

      <div className="flex-1 space-y-2 overflow-y-auto p-4">
        {messages?.length === 0 && <p className="text-muted text-sm">{t("chat.noMessages")}</p>}
        {messages?.map((message) => (
          <MessageBubble key={message.id} message={message} own={message.sender_id === user?.id} />
        ))}
      </div>

      {view.canDecide && (
        <div className="border-line space-y-2 border-t p-4">
          <p className="text-muted text-sm">
            {t("chat.wantsToChat", {
              username: other.username,
              listing: conversation.listing_title,
            })}
          </p>
          <div className="flex gap-2">
            <Button
              variant="primary"
              disabled={accept.isPending || decline.isPending}
              onClick={() => void decide("accept")}
            >
              {t("chat.accept")}
            </Button>
            <Button
              variant="secondary"
              disabled={accept.isPending || decline.isPending}
              onClick={() => void decide("decline")}
            >
              {t("chat.decline")}
            </Button>
          </div>
        </div>
      )}

      <form onSubmit={(e) => void handleSend(e)} className="border-line border-t p-4">
        {view.sendDisabledKey ? (
          <p className="text-muted text-sm">{t(view.sendDisabledKey)}</p>
        ) : (
          <div className="flex gap-2">
            <input
              aria-label={t("chat.messageLabel")}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder={t("chat.placeholder")}
              className="border-line bg-surface text-foreground flex-1 rounded-md border px-3 py-2"
            />
            <Button type="submit" variant="primary" disabled={!draft.trim() || send.isPending}>
              {send.isPending ? t("common.saving") : t("chat.send")}
            </Button>
          </div>
        )}

        {error && (
          <p role="alert" className="text-danger mt-2 text-sm">
            {error}
          </p>
        )}
      </form>
    </div>
  );
}

function MessageBubble({ message, own }: { message: Message; own: boolean }) {
  return (
    <div className={`flex ${own ? "justify-end" : "justify-start"}`}>
      <p
        className={`max-w-[75%] rounded-lg px-3 py-2 text-sm break-words ${
          own ? "bg-accent text-accent-contrast" : "bg-surface-muted text-foreground"
        }`}
      >
        {message.body}
      </p>
    </div>
  );
}
