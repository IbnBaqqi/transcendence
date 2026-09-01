import { useTranslation } from "react-i18next";

import { supportedLngs, type Locale } from "../../i18n";

export function LanguageSwitcher() {
  const { i18n, t } = useTranslation();
  const current = (i18n.language?.split("-")[0] ?? "en") as Locale;

  return (
    <select
      aria-label={t("language.label")}
      value={supportedLngs.includes(current) ? current : "en"}
      onChange={(e) => void i18n.changeLanguage(e.target.value)}
      className="text-muted hover:text-foreground cursor-pointer border-0 bg-transparent text-sm focus:outline-none"
    >
      {supportedLngs.map((lng) => (
        <option key={lng} value={lng}>
          {t(`language.${lng}`)}
        </option>
      ))}
    </select>
  );
}
