import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

import Avatar from "./Avatar";
import { FollowButton } from "./FollowButton";
import { PresenceIndicator } from "./PresenceIndicator";
import { Skeleton } from "./Skeleton";
import { useListing } from "../../api/listings";
import { usePublicProfile } from "../../api/profile";
import { deriveInitials } from "../../lib/initials";

export function SellerFollowSection({ listingId }: { listingId: string }) {
  const { t } = useTranslation();
  const { data: listing, isPending } = useListing(listingId);
  const { data: seller } = usePublicProfile(listing?.seller_id);

  if (!listingId) return null;
  if (isPending) return <Skeleton className="h-16 w-full" />;
  if (!listing || !seller) return null;

  return (
    <div className="border-line bg-surface flex items-center gap-3 rounded-lg border p-3">
      <Avatar
        size="sm"
        initials={deriveInitials(seller.username)}
        imageUrl={seller.avatar_url ?? undefined}
      />
      <div className="min-w-0 flex-1">
        <p className="text-muted text-xs">{t("listings.seller")}</p>
        <Link to={`/users/${seller.id}`} className="text-foreground font-medium hover:underline">
          {seller.username}
        </Link>
        {seller.presence && <PresenceIndicator presence={seller.presence} />}
      </div>
      <FollowButton userId={seller.id} />
    </div>
  );
}
