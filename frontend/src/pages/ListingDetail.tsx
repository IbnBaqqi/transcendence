import { useParams } from "react-router-dom";

// stub for #21
//
// useParams reads the dynamic segments of the current URL. The route is
// registered as "/listings/:id", so the ":id" part is captured by name -
// visiting /listings/7 gives { id: "7" }.
//
// two thing worth knowing: the value is always a string (URLs are text), so
// it needs Number() before being used as a numeric id. and its typed as
// string | undefined, because TypeScript can't know which route rendered this
// component - React Router has no way to prove ":id" exists in the path
export default function ListingDetail() {
  const { id } = useParams();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">Listing {id}</h1>
      <p className="text-muted mt-2">
        {/* TODO(#21): image gallery, full description, seller info, price,
				 	buy/contact CTA. hardcode a Listing from ../api/types until the
					mock layer (#79) lands. */}
        Listing detail page - not built yet (#21).
      </p>
    </div>
  );
}
