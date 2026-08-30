import { useListings } from "../api/listings";
import { useCategoryNames } from "../api/categories";
import { ListingCard } from "../components/objects/ListingCard";
import { Skeleton } from "../components/objects/Skeleton";

export default function Home() {
  // useListings() gives us the query's state: the data plus loading/error flags.
  const { data: listings, isPending, isError, refetch } = useListings();
  const categoryName = useCategoryNames();

  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">Latest Finds</h1>
      {/*
        Live region: always in the DOM so assistive tech has something to
        watch. When the text inside changes, screen readers announce it.
        Rendering this <p> conditionally would defeat that - the region and its
        content would appear together, which often goes unannounced.
        aria-live="polite" = wait for a pause, don't interrupt the user.
        When every condition is false it renders empty, taking no space.
      */}
      {/* Plain text Loading / Error / Message view */}
      {/* <p role="status" className="text-muted mt-4"> */}
      {/*   {isPending && "Loading..."} */}
      {/*   {isError && "Couldn't load listings. Try again."} */}
      {/*   {listings?.length === 0 && "No listings yet!"} */}
      {/* </p> */}
      {isPending && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-56 w-full" />
          ))}
        </div>
      )}
      {isError && (
        <Skeleton
          variant="error"
          className="mt-6 h-56 w-full"
          message="Couldn't load listings."
          onRetry={() => refetch()}
        />
      )}
      {listings?.length === 0 && (
        <Skeleton
          variant="error"
          className="mt-6 h-56 w-full"
          message="No listings yet."
          onRetry={() => refetch()}
        />
      )}
      {listings && listings.length > 0 && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {listings.map((listing) => (
            <ListingCard
              key={listing.id}
              listing={listing}
              categoryName={categoryName(listing.category)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
