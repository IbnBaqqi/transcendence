import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

// stub for #21
//
// useParams reads the dynamic segments of the current URL. The route is
// registered as "/listings/:id", so the ":id" part is captured by name -
// visiting /listings/01a02305-b81c-7dcb-86a0-7f75e33e0af3 gives that
// string back under { id }.
//
// two things worth knowing: the value is always a string (URLs are text),
// which is exactly what the API wants - ids are uuids, so it goes straight
// into a request with no conversion. and it's typed string | undefined,
// because TypeScript can't know which route rendered this component - React
// Router has no way to prove ":id" exists in the path
export default function ListingDetail() {
  const { id } = useParams();
  const { t } = useTranslation();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">
        {t("pages.listingDetail.title", { id })}
      </h1>
      <p className="text-muted mt-2">
        {/* TODO(#21): image gallery, full description, seller info, price,
				 	buy/contact CTA. hardcode a Listing from ../api/types until the
					mock layer (#79) lands. */}
        {t("pages.listingDetail.stub")}
      </p>
    </div>
  );
}
