import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import Button from "./Button";
import { isApiError } from "../../api/client";
import { keys } from "../../api/queryKeys";
import {
  useCancelOrder,
  useConfirmOrder,
  useHandoverOrder,
  useReceiveOrder,
} from "../../api/orders";
import { deriveOrderView, type OrderAction } from "../../lib/orderState";
import { useAuth } from "../../hooks/useAuth";
import type { Order } from "../../api/types";

const LABELS: Record<OrderAction, { idle: string; busy: string }> = {
  confirm: { idle: "Confirm reservation", busy: "Confirming…" },
  handover: { idle: "Mark handed over", busy: "Marking…" },
  receive: { idle: "Confirm receipt", busy: "Confirming…" },
  cancel: { idle: "Cancel order", busy: "Cancelling…" },
};

export function OrderActions({ order }: { order: Order }) {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  // All four run every render - hooks can't be called conditionally, so we
  // build the whole map and let the view decide which buttons appear.
  const mutations: Record<OrderAction, ReturnType<typeof useConfirmOrder>> = {
    confirm: useConfirmOrder(),
    handover: useHandoverOrder(),
    receive: useReceiveOrder(),
    cancel: useCancelOrder(),
  };

  const view = deriveOrderView(order, user?.id);
  const busy = Object.values(mutations).some((m) => m.isPending);

  async function run(action: OrderAction) {
    setError(null);
    try {
      await mutations[action].mutateAsync(order.id);
    } catch (err) {
      // 409 means the order moved under us - the other side acted first, or a
      // second tab did. Refetch so the buttons redraw rather than insisting.
      if (isApiError(err) && err.status === 409) {
        void qc.invalidateQueries({ queryKey: keys.orders.all });
        setError("This order just changed. Refreshed - check the actions again.");
        return;
      }
      // 403 lands here: the backend's message names the rule we got wrong.
      setError(isApiError(err) ? err.message : "Something went wrong. Please try again.");
    }
  }

  if (view.actions.length === 0 && error === null) return null;

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-2">
        {view.actions.map((action) => (
          <Button
            key={action}
            variant={action === "cancel" ? "secondary" : "primary"}
            disabled={busy}
            onClick={() => void run(action)}
          >
            {mutations[action].isPending ? LABELS[action].busy : LABELS[action].idle}
          </Button>
        ))}
      </div>
      {error && (
        <p role="alert" className="text-berry-500 text-sm">
          {error}
        </p>
      )}
    </div>
  );
}
