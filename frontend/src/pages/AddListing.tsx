import { AddListingSection } from "../components/forms/AddListingSection";
import { useTranslation } from "react-i18next";

export default function AddListing() {
  const { t } = useTranslation();
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.addListing.title")}</h1>
      <AddListingSection />
    </div>
  );
}
