import { useState } from "react";

import { useConversations } from "../../api/conversations";
import { isApiError } from "../../api/client";
import { useAuth } from "../../hooks/useAuth";
import { useModal } from "../../providers/modalContext";
import { useTranslation } from "react-i18next";

import { ConversationList } from "../chat/ConversationList";
import { MessageThread } from "../chat/MessageThread";
import Button from "../objects/Button.tsx";
import { Skeleton } from "../objects/Skeleton";

export function Chat() {
  const { t } = useTranslation();
  const { closeChat, openModal, chatConversationId } = useModal();
  const { user } = useAuth();

  // ChatRoot unmounts this when the modal closes, so the initialiser runs on
  // every open - which is what makes "open straight into this thread" work.
  const [selectedId, setSelectedId] = useState<string | null>(chatConversationId);

  const { data: conversations, isPending, isError, error, refetch } = useConversations(!!user);

  return (
    <div className="bg-surface-muted flex h-full flex-col">
      <div className="border-line flex items-center justify-between border-b px-4 py-3">
        <h2 className="text-foreground text-lg font-semibold">{t("chat.title")}</h2>
        <Button variant="secondary" type="button" onClick={closeChat}>
          {t("chat.close")}
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {!user ? (
          <div className="space-y-3 p-4">
            <p className="text-muted text-sm">{t("chat.signedOut")}</p>
            <Button variant="primary" onClick={() => openModal("login")}>
              {t("common.logIn")}
            </Button>
          </div>
        ) : selectedId ? (
          <MessageThread conversationId={selectedId} onBack={() => setSelectedId(null)} />
        ) : isPending ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        ) : isError ? (
          <Skeleton
            variant="error"
            className="m-4 h-24"
            message={isApiError(error) ? error.message : t("chat.listError")}
            onRetry={() => refetch()}
          />
        ) : (
          <ConversationList conversations={conversations ?? []} onSelect={setSelectedId} />
        )}
      </div>
    </div>
  );
}
