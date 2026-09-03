import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, apiPath } from "./client";
import { keys } from "./queryKeys";
import type {
  ModerateListingInput,
  ModerateListingResponse,
  ModerationAction,
  Report,
  ReportedListing,
} from "./types";

export function useReportQueue() {
  return useQuery({
    queryKey: keys.moderation.queue(),
    queryFn: async () => (await api.get<ReportedListing[]>("/admin/reports")).data ?? [],
  });
}

export function useListingReports(listingId: string) {
  return useQuery({
    queryKey: keys.moderation.reports(listingId),
    queryFn: async () =>
      (await api.get<Report[]>(apiPath`/admin/listings/${listingId}/reports`)).data ?? [],
    enabled: listingId !== "",
  });
}

export function useModerationHistory(listingId: string) {
  return useQuery({
    queryKey: keys.moderation.history(listingId),
    queryFn: async () =>
      (await api.get<ModerationAction[]>(apiPath`/admin/listings/${listingId}/moderation`)).data ??
      [],
    enabled: listingId !== "",
  });
}

export function useModerateListing() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: async ({ listingId, ...input }: ModerateListingInput & { listingId: string }) =>
      (
        await api.post<ModerateListingResponse>(
          apiPath`/admin/listings/${listingId}/moderate`,
          input,
        )
      ).data,

    // One action resolves every open report on the listing and writes an audit
    // row, so all three moderation keys are stale - not just the queue. The
    // listing itself changes visibility, which is what listings.all covers -
    // and by prefix that already includes its detail entry.
    onSuccess: () =>
      Promise.all([
        qc.invalidateQueries({ queryKey: keys.moderation.all }),
        qc.invalidateQueries({ queryKey: keys.listings.all }),
      ]),

    // Every failure here means this copy of the moderation data is out of date:
    // a 409 is usually another moderator getting there first, and a 404 is the
    // listing being gone. Their decision also changed the reports and history
    // sitting open on screen, so this covers all three rather than the queue
    // alone - and the collapsed rows' queries are disabled, so it is cheap.
    onError: () => qc.invalidateQueries({ queryKey: keys.moderation.all }),
  });
}
