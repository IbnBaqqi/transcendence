import { useTranslation } from "react-i18next";

import { AddListingSection } from "../components/forms/AddListingSection";
import Button from "../components/objects/Button";
import { useAuth } from "../hooks/useAuth";
import { useModal } from "../providers/modalContext";

export default function AddListing() {
  const { t } = useTranslation();
  const { user, isLoading } = useAuth();
  const { openModal } = useModal();

  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.addListing.title")}</h1>
      {isLoading ? (
        <p className="text-muted text-sm">{t("common.loading")}</p>
      ) : user ? (
        <AddListingSection />
      ) : (
        <div className="space-y-3">
          <p className="text-muted text-sm">{t("forms.addListing.signedOut")}</p>
          <Button variant="primary" onClick={() => openModal("login")}>
            {t("common.logIn")}
          </Button>
        </div>
      )}
    </div>
  );
}
