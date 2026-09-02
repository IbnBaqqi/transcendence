import { useParams } from "react-router-dom";

import { ReserveListingSection } from "../components/forms/ReserveListingSection";

// Still the #21 stub. The reserve box is self-contained, so whoever builds the
// real page moves one line rather than untangling the order flow from it.
export default function ListingDetail() {
  const { id } = useParams();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">Listing {id}</h1>
      <p className="text-muted mt-2">
        {/* TODO(#21): image gallery, full description, seller info. */}
        Listing detail page - not built yet (#21).
      </p>
      <div className="mt-6">
        <ReserveListingSection listingId={id ?? ""} />
      </div>
    </div>
  );
}
