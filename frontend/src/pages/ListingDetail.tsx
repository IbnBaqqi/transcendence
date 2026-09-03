import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { ReserveListingSection } from "../components/forms/ReserveListingSection";
import { StartConversationSection } from "../components/forms/StartConversationSection";
import { SellerFollowSection } from "../components/objects/SellerFollowSection";

// Still the #21 stub. The reserve box is self-contained, so whoever builds the
// real page moves one line rather than untangling the order flow from it.
export default function ListingDetail() {
  const { id } = useParams();
  const { t } = useTranslation();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">
        {t("pages.listingDetail.title", { id })}
      </h1>
      <p className="text-muted mt-2">
        {/* TODO(#21): image gallery, full description, seller info. */}
        {t("pages.listingDetail.stub")}
      </p>
      <div className="mt-6 space-y-4">
        <SellerFollowSection listingId={id ?? ""} />
        <ReserveListingSection listingId={id ?? ""} />
        <StartConversationSection listingId={id ?? ""} />
      </div>
    </div>
  );
}
