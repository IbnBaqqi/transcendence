import { useTranslation } from "react-i18next";

export default function AdminListings() {
  const { t } = useTranslation();

  return (
    <div className="mx-auto max-w-4xl px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.adminListings.title")}</h1>
    </div>
  );
}
