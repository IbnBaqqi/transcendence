import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

// rendered by "*" catch-all route for any unknown URL
export default function NotFound() {
  const { t } = useTranslation();
  return (
    <div className="max-w-column mx-auto px-4 py-16 text-center">
      <h1 className="text-foreground text-page-title font-bold">{t("pages.notFound.title")}</h1>
      <p className="text-muted mt-2">{t("pages.notFound.subtitle")}</p>
      <Link to="/" className="text-accent mt-4 inline-block hover:underline">
        {t("pages.notFound.backHome")}
      </Link>
    </div>
  );
}
