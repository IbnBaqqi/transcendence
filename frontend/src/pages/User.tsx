// This is a stub for #105
//
// Currently accessible at localhost:5173/user but needs to exist at a unique
// URL for each user

// Imports for unique URLs when we add them
// import { useParams } from "react-router-dom";
// import { useEffect, useState } from "react";
// import NotFound from "../pages/NotFound";
import Avatar from "../components/objects/Avatar.tsx";
import Mailto from "../components/objects/Mailto.tsx";
import { useListings } from "../api/listings";
import { ListingCard } from "../components/objects/ListingCard";
import Button from "../components/objects/Button.tsx";
import { useModal } from "../providers/modalContext";

export default function User() {
  {
    /* Placeholder, needs to pull only user's listings */
  }
  const { openChat } = useModal();
  const { data: listings, isPending, isError } = useListings();
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">User Profile</h1>
      <div className="flex flex-row gap-4">
        <div>
          <Avatar size="lg" initials="OR" />
        </div>
        <div className="text-accent my-auto flex flex-col text-base">
          <div className="font-bold">Oscar Rogers</div>
          {/* Hide email depending on preferences */}
          <Mailto label="oscarroff@example.com" mailto="oscarrogers@example.com" />
        </div>
      </div>
      {/* If user profile does not match logged in entity then display button */}
      <Button variant="secondary" onClick={() => openChat()}>
        Message User
      </Button>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Contact Details</h2>
        <div className="flex flex-row gap-4">
          <div className="flex flex-col">
            <div className="text-muted">First Name</div>
            <div>Oscar</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Last Name</div>
            <div>Rogers</div>
          </div>
          {/* Hide remaining details depending on preferences */}
          <div className="flex flex-col">
            <div className="text-muted">Telephone</div>
            <div>1234567890</div>
          </div>
          <div className="flex flex-col">
            <div className="text-muted">Location</div>
            {/* Open Maps integration? */}
            <div>Helsinki</div>
          </div>
        </div>
        <div className="flex flex-row gap-2"></div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Bio</h2>
        <div className="">
          Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt
          ut labore et dolore magna aliqua. Fusce auctor at fringilla aliquam massa iaculis et ad
          potenti cras purus. Curabitur himenaeos maximus viverra iaculis consectetur a enim. Magna
          in augue viverra primis aenean magna magna donec et quisque hendrerit etiam. Ullamcorper
          fames varius elementum sagittis elementum vitae eu inceptos quam imperdiet. A conubia
          aliquet libero molestie ultricies sagittis quam nostra cubilia elementum amet porta.
        </div>
      </div>
      <div className="space-y-1">
        <h2 className="text-foreground text-lg font-bold">Listings</h2>
        {/* Placeholder, format, length etc. to be decided later */}
        <p role="status" aria-live="polite" className="text-muted mt-4">
          {isPending && "Loading..."}
          {isError && "Couldn't load listings. Try again."}
          {listings?.length === 0 && "No listings yet!"}
        </p>
        {listings && listings.length > 0 && (
          <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {listings.map((listing) => (
              <ListingCard key={listing.id} listing={listing} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
