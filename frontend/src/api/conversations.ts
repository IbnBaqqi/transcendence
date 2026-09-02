import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type { Conversation, ConversationListItem, Message } from "./types";

// Demo-scale polling, not a considered production number: no WebSockets in
// scope (#88), so an open thread re-asks the server on a timer.
const THREAD_POLL_MS = 5_000;
const LIST_POLL_MS = 30_000;

export interface StartConversationInput {
  listing_id: string;
  body: string; // the opening message ships with the request
}

// --- Reads ---

export function useConversations(enabled = true) {
  return useQuery({
    queryKey: keys.conversations.list(),
    queryFn: async () => (await api.get<ConversationListItem[]>("/conversations")).data ?? [],
    refetchInterval: LIST_POLL_MS,
    enabled,
  });
}

export function useConversation(id: string) {
  return useQuery({
    queryKey: keys.conversations.detail(id),
    queryFn: async () => (await api.get<Conversation>(apiPath`/conversations/${id}`)).data,
    enabled: id !== "",
  });
}

export function useMessages(id: string) {
  return useQuery({
    // The API answers newest-first (ORDER BY id DESC); reverse once here so
    // every consumer gets oldest-at-top and none of them re-sorts.
    queryKey: keys.conversations.messages(id),
    queryFn: async () =>
      ((await api.get<Message[]>(apiPath`/conversations/${id}/messages`)).data ?? [])
        .slice()
        .reverse(),
    enabled: id !== "",
    refetchInterval: THREAD_POLL_MS,
  });
}

// --- Writes ---

export function useStartConversation() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (input: StartConversationInput) =>
      (await api.post<Conversation>("/conversations", input)).data,

    onSuccess: (conversation) => {
      qc.setQueryData(keys.conversations.detail(conversation.id), conversation);
      qc.invalidateQueries({ queryKey: keys.conversations.list() });
    },
  });
}

export function useSendMessage(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (body: string) =>
      (await api.post<Message>(apiPath`/conversations/${id}/messages`, { body })).data,

    onSuccess: () => {
      // Refetch rather than appending the response: the poll may already have
      // pulled it in, and two sources writing the same list is how duplicates
      // appear.
      qc.invalidateQueries({ queryKey: keys.conversations.messages(id) });
      qc.invalidateQueries({ queryKey: keys.conversations.list() });
    },
  });
}

// Accept and decline differ only by path, and both answer with the updated
// conversation.
function useConversationDecision(decision: "accept" | "decline") {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async (id: string) =>
      (await api.post<Conversation>(apiPath`/conversations/${id}/${decision}`)).data,

    onSuccess: (conversation) => {
      qc.setQueryData(keys.conversations.detail(conversation.id), conversation);
      qc.invalidateQueries({ queryKey: keys.conversations.list() });
    },
  });
}

export const useAcceptConversation = () => useConversationDecision("accept");
export const useDeclineConversation = () => useConversationDecision("decline");

export function useMarkConversationRead() {
  const qc = useQueryClient();

  return useMutation({
    // 204, so there is no body to read or cache.
    mutationFn: async (id: string) => {
      await api.post(apiPath`/conversations/${id}/read`);
    },

    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.conversations.list() });
      // The header badge reads this one.
      qc.invalidateQueries({ queryKey: keys.me.unread() });
    },
  });
}
