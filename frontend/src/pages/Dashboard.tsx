// Stub for #23.
//
// Nothing behind this page exists yet, so it's a visual pass: hardcode local
// fixtures. Listings can reuse the `Listing` type from ../api/types, but
// orders and payouts have no types - keep those shapes local to this file.
// Don't add an `Order` type to api/types.ts before #13 defines what an order
// actually is; a guessed type in the shared contract is worse than none.
//
// Conceptually this page is behind auth, but protected routes are #46 and
// backend auth is unfinished - don't try to gate it yet.
export default function Dashboard() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">Seller Dashboard</h1>
      <p className="text-muted mt-2">
        {/* TODO(#23): active listings, pending orders (#13), recent payouts,
            quick actions (manage listing / confirm order) as inert buttons. */}
        Seller dashboard - not built yet (#23).
      </p>
    </div>
  );
}
