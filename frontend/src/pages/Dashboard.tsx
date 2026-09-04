import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useOrders } from "../api/orders";
import { useSearchListings } from "../api/listings";
import { isApiError } from "../api/client";
import { useAuth } from "../hooks/useAuth";
import { useModal } from "../providers/modalContext";
import { deriveListingStats, groupOrdersByUrgency } from "../lib/sellerStats";
import { OrderCard } from "../components/objects/OrderCard";
import { SellerListingRow } from "../components/objects/SellerListingRow";
import Button from "../components/objects/Button";
import { Skeleton } from "../components/objects/Skeleton";
import { Pagination } from "../components/objects/Pagination";
import { usePageParam } from "../hooks/usePageParam";
import { PAGE_SIZE } from "../lib/searchFilters";

export default function Dashboard() {
  const { t } = useTranslation();
  const { user, isLoading: authLoading } = useAuth();
  const { openModal } = useModal();

  const [page, setPage] = usePageParam();
  const listingsQuery = useSearchListings(
    { seller_id: user?.id, page, limit: PAGE_SIZE, include_sold_out: true },
    { enabled: !!user },
  );
  const ordersQuery = useOrders({ enabled: !!user });

  if (authLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (!user) {
    return (
      <div className="mx-auto max-w-4xl space-y-3 px-4 py-8">
        <h1 className="text-foreground text-page-title font-bold">{t("pages.dashboard.title")}</h1>
        <p className="text-muted text-sm">{t("pages.dashboard.signedOut")}</p>
        <Button variant="primary" onClick={() => openModal("login")}>
          {t("common.logIn")}
        </Button>
      </div>
    );
  }

  const listings = listingsQuery.data?.items ?? [];
  // Sales only: GET /orders carries this user's purchases too, and those
  // belong on /orders, not on the seller's dashboard.
  const sales = ordersQuery.data?.filter((o) => o.seller_id === user.id) ?? [];
  const groups = groupOrdersByUrgency(sales, user.id);
  const allOrders = ordersQuery.data ?? [];

  return (
    <div className="mx-auto max-w-4xl space-y-8 px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.dashboard.title")}</h1>

      {groups.needsYou.length > 0 && (
        <Section title={t("pages.dashboard.needsYou", { count: groups.needsYou.length })}>
          {groups.needsYou.map((order) => (
            <OrderCard key={order.id} order={order} />
          ))}
        </Section>
      )}

      <Section title={t("pages.dashboard.myListings")}>
        {listingsQuery.isPending ? (
          <Skeleton className="h-20 w-full" />
        ) : listingsQuery.isError ? (
          <Skeleton
            variant="error"
            className="h-20 w-full"
            message={
              isApiError(listingsQuery.error)
                ? listingsQuery.error.message
                : t("pages.dashboard.listingsError")
            }
            onRetry={() => listingsQuery.refetch()}
          />
        ) : listingsQuery.data?.total === 0 ? (
          <div className="border-line rounded-lg border border-dashed p-8 text-center">
            <p className="text-muted text-sm">{t("pages.dashboard.noListings")}</p>
            <Link
              to="/addlisting"
              className="text-accent mt-2 inline-block text-sm hover:underline"
            >
              {t("pages.dashboard.addListing")}
            </Link>
          </div>
        ) : (
          <>
            {listings.map((listing) => (
              <SellerListingRow
                key={listing.id}
                listing={listing}
                // Every order, not just sales: a listing's stock is spent by
                // whoever ordered it.
                stats={deriveListingStats(listing, allOrders)}
              />
            ))}
            {listingsQuery.data && (
              <div className="mt-4">
                <Pagination
                  page={listingsQuery.data.page}
                  totalPages={listingsQuery.data.total_pages}
                  total={listingsQuery.data.total}
                  onPageChange={setPage}
                />
              </div>
            )}
          </>
        )}
      </Section>

      {groups.inProgress.length > 0 && (
        <Section title={t("pages.dashboard.inProgress", { count: groups.inProgress.length })}>
          {groups.inProgress.map((order) => (
            <OrderCard key={order.id} order={order} />
          ))}
        </Section>
      )}

      {groups.finished.length > 0 && (
        <Section title={t("pages.dashboard.finished", { count: groups.finished.length })}>
          {groups.finished.map((order) => (
            <OrderCard key={order.id} order={order} />
          ))}
        </Section>
      )}

      {ordersQuery.isError && (
        <Skeleton
          variant="error"
          className="h-20 w-full"
          message={t("pages.dashboard.ordersError")}
          onRetry={() => ordersQuery.refetch()}
        />
      )}

      {!ordersQuery.isPending && !ordersQuery.isError && sales.length === 0 && (
        <p className="text-muted text-sm">{t("pages.dashboard.noOrders")}</p>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-3">
      <h2 className="text-foreground text-section font-semibold">{title}</h2>
      {children}
    </section>
  );
}
