// This is a stub for whoever picks up #26
import { useTranslation } from "react-i18next";

export default function Notifications() {
  const { t } = useTranslation();
  return (
    <div className="mx-auto max-w-3xl space-y-5 px-4 py-8">
      <h1 className="text-foreground text-3xl font-bold">{t("pages.notifications.title")}</h1>
    </div>
  );
}
