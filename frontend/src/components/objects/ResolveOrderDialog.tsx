import { useState } from "react";
import { useTranslation } from "react-i18next";

import { useResolveOrder } from "../../api/adminOrders";
import { isApiError } from "../../api/client";
import Button from "./Button";
import type { AdminOrder, ResolveOutcome } from "../../api/types";

const REASON_MAX = 500;

// stuck is handshake_stuck OR stranded (migration 016). handshake_stuck needs
// exactly one handshake mark; stranded needs neither. So a stuck order with
// neither mark can only be the stranded shape - which the server will accept
// as cancelled alone, because completed would assert a handover that never
// happened and refunded implies a trade that got far enough to unwind.
//
// Derivable here even though stranded also requires both accounts deleted,
// which AdminOrder does not say: stuck is what rules the other shape out.
function legalOutcomes(order: AdminOrder): ResolveOutcome[] {
  const neitherActed = order.seller_handed_over_at === null && order.buyer_received_at === null;
  return neitherActed ? ["cancelled"] : ["completed", "cancelled", "refunded"];
}

export function ResolveOrderDialog({ order }: { order: AdminOrder }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [outcome, setOutcome] = useState<ResolveOutcome | null>(null);
  const [reason, setReason] = useState("");
  const resolve = useResolveOrder();

  // Only a stuck order can be resolved; offering the button otherwise is how
  // you generate 409s deliberately.
  if (!order.stuck) return null;

  const outcomes = legalOutcomes(order);
  const trimmed = reason.trim();
  const canSubmit = outcome !== null && trimmed !== "";

  const done = () => {
    setOpen(false);
    setOutcome(null);
    setReason("");
    // The alert is derived from the mutation and rendered outside the open
    // branch, so clearing only the form leaves a 409 under a collapsed row
    // with nothing to explain it. Reachable on "that shape only takes
    // cancelled", where the order stays stuck and the row stays mounted.
    resolve.reset();
  };

  function submit() {
    if (!canSubmit || outcome === null) return;
    resolve.mutate({ orderId: order.id, outcome, reason: trimmed }, { onSuccess: done });
  }

  return (
    <div className="border-line mt-3 border-t pt-3">
      {!open ? (
        <Button variant="secondary" onClick={() => setOpen(true)}>
          {t("adminOrders.resolve")}
        </Button>
      ) : (
        <div className="space-y-2">
          <div className="flex flex-wrap gap-2">
            {outcomes.map((option) => (
              <Button
                key={option}
                variant={outcome === option ? "primary" : "secondary"}
                onClick={() => setOutcome(option)}
              >
                {t(`adminOrders.outcome.${option}`)}
              </Button>
            ))}
          </div>

          {outcomes.length === 1 && (
            <p className="text-muted text-sm">{t("adminOrders.onlyCancellable")}</p>
          )}

          <label className="text-muted block text-sm" htmlFor={`resolve-reason-${order.id}`}>
            {/* It lands in the order's history as the note, and both parties
                can read it - so the label says who it is written for. */}
            {t("adminOrders.reasonForParties")}
          </label>
          <textarea
            id={`resolve-reason-${order.id}`}
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            maxLength={REASON_MAX}
            rows={2}
            className="border-line bg-surface text-foreground w-full rounded border p-2 text-sm"
          />

          <div className="flex justify-end gap-2">
            <Button variant="tertiary" onClick={done}>
              {t("common.cancel")}
            </Button>
            <Button variant="primary" onClick={submit} disabled={!canSubmit || resolve.isPending}>
              {resolve.isPending ? t("adminOrders.resolving") : t("adminOrders.confirmResolve")}
            </Button>
          </div>
        </div>
      )}

      {/* A 409 is "this order is not stuck any more" or "that shape only takes
          cancelled" - only the server's own message tells them apart. The list
          refetches either way; that is in the mutation. */}
      {resolve.isError && (
        <p role="alert" className="text-berry-500 mt-2 text-sm">
          {isApiError(resolve.error) ? resolve.error.message : t("adminOrders.resolveFailed")}
        </p>
      )}
    </div>
  );
}
