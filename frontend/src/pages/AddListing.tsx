import { AddListingSection } from "../components/forms/AddListingSection";

export default function AddListing() {
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">Add Listing</h1>
      <AddListingSection />
    </div>
  );
}
