import { useTranslation } from "react-i18next";

export default function Dashboard() {
  const { t } = useTranslation();
  return (
    <div className="mx-auto max-w-6xl px-4 py-8">
      <h1 className="text-foreground text-2xl font-bold">{t("pages.dashboard.title")}</h1>
      <p className="text-muted mt-2">
        {/* TODO(#23): active listings, pending orders (#13), recent payouts,
            quick actions (manage listing / confirm order) as inert buttons. */}
        {t("pages.dashboard.stub")}
      </p>
    </div>
  );
}
