import { useState } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useOrders } from "../api/orders";
import { isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useModal } from "../providers/modalContext";
import { OrderCard } from "../components/objects/OrderCard";
import Button from "../components/objects/Button";
import { Skeleton } from "../components/objects/Skeleton";
import type { Order } from "../api/types";

type Tab = "purchases" | "sales";

// GET /orders returns purchases and sales in one list and promises no order.
function byNewest(a: Order, b: Order) {
  return b.created_at.localeCompare(a.created_at);
}

export default function Orders() {
  const { user, isLoading: authLoading } = useAuth();
  const { t } = useTranslation();
  const { openModal } = useModal();
  const [tab, setTab] = useState<Tab>("purchases");

  const { data: orders, isPending, isError, error, refetch } = useOrders({ enabled: !!user });

  if (authLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (!user) {
    return (
      <div className="mx-auto max-w-4xl space-y-3 px-4 py-8">
        <h1 className="text-foreground text-2xl font-bold">{t("orders.title")}</h1>
        <p className="text-muted text-sm">{t("orders.signedOut")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  const purchases = orders?.filter((o) => o.buyer_id === user.id).sort(byNewest) ?? [];
  const sales = orders?.filter((o) => o.seller_id === user.id).sort(byNewest) ?? [];
  const shown = tab === "purchases" ? purchases : sales;

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{t("orders.title")}</h1>

      <div className="border-line mt-4 flex gap-2 border-b">
        <TabButton active={tab === "purchases"} onClick={() => setTab("purchases")}>
          {/* No count until the list arrives - "(0)" next to a skeleton is a
              claim we can't make yet. */}
          {t("orders.tabs.buying")}
          {orders && ` (${purchases.length})`}
        </TabButton>
        <TabButton active={tab === "sales"} onClick={() => setTab("sales")}>
          {t("orders.tabs.selling")}
          {orders && ` (${sales.length})`}
        </TabButton>
      </div>

      {isPending && (
        <div className="mt-6 space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 w-full" />
          ))}
        </div>
      )}

      {isError && (
        <Skeleton
          variant="error"
          className="mt-6 h-28 w-full"
          message={isApiError(error) ? error.message : t("orders.listError")}
          onRetry={() => refetch()}
        />
      )}

      {orders && shown.length === 0 && (
        <div className="border-line mt-6 rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted text-sm">
            {tab === "purchases" ? t("orders.empty.purchases") : t("orders.empty.sales")}
          </p>
          {tab === "purchases" && (
            <Link to="/search" className="text-accent mt-2 inline-block text-sm hover:underline">
              {t("orders.empty.browse")}
            </Link>
          )}
        </div>
      )}

      {shown.length > 0 && (
        <div className="mt-6 space-y-3">
          {shown.map((order) => (
            <OrderCard key={order.id} order={order} />
          ))}
        </div>
      )}
    </div>
  );
}

// aria-pressed rather than a real tablist: no arrow-key handling to get wrong.
function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`-mb-px border-b-2 px-3 py-2 text-sm font-medium ${
        active
          ? "border-accent text-foreground"
          : "text-muted hover:text-foreground border-transparent"
      }`}
    >
      {children}
    </button>
  );
}
