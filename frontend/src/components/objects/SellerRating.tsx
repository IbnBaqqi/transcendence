import { useTranslation } from "react-i18next";

import type { Rating } from "../../api/types";

// Below this a seller is "new" rather than badly rated. The server does not
// know this number and does not need to: it sorts unrated sellers last with
// COALESCE, so the threshold is purely how much evidence we think is worth
// showing an average for.
const ENOUGH_REVIEWS = 5;

export function SellerRating({ rating }: { rating: Rating }) {
  const { t } = useTranslation();

  // count, not average: an unrated seller is {average: 0, count: 0}, so
  // reading the average alone brands everyone new as a zero-star seller.
  if (rating.count < ENOUGH_REVIEWS) {
    return <span className="text-muted text-xs">{t("listings.newSeller")}</span>;
  }

  return (
    <span className="text-muted text-xs">
      {/* One decimal: the average is a float and 4.333333 is noise, not detail. */}
      {t("listings.rating", { average: rating.average.toFixed(1), count: rating.count })}
    </span>
  );
}
