// footer links don't need active styling so only Link is needed
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";

export default function Footer() {
  const { t } = useTranslation();

  return (
    <footer className="border-line bg-surface border-t">
      {/* sm is meant for wider screens */}
      <div className="text-muted mx-auto flex max-w-6xl flex-col gap-2 px-4 py-6 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p>
          © {new Date().getFullYear()} {t("brand")}
        </p>
        <nav className="flex gap-4">
          <Link to="/privacy" className="hover:text-foreground">
            {t("footer.privacyPolicy")}
          </Link>
          <Link to="/terms" className="hover:text-foreground">
            {t("footer.termsOfService")}
          </Link>
        </nav>
      </div>
    </footer>
  );
}
