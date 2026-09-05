import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { LanguageSwitcher } from "../objects/LanguageSwitcher";
import { NotificationBell } from "../objects/NotificationBell";
import { UserMenu } from "./UserMenu";

// inline-flex, not the default inline: an inline <svg> sits on the text
// baseline and rides low next to its neighbours.
const iconLink = "text-muted hover:text-foreground inline-flex items-center";
const iconSize = "xs:h-6 xs:w-5 h-5 w-4";

export default function Header() {
  const { t } = useTranslation();

  // z-50 governs the menus below too: a sticky element with a z-index makes the
  // header one stacking context, so their own z-index cannot lift them out of
  // it. Above the chat panel, below the dialog backdrop.
  return (
    <header className="border-line bg-surface sticky top-0 z-50 border-b">
      <div className="max-w-wide mx-auto flex items-center justify-between px-4 py-3">
        {/* One link, not two: the mark and the wordmark are the same target
            with the same name, and as separate links they were two tab stops
            announcing "Metsatori" twice. The label is on the link because the
            wordmark is display:none below sm and the mark is aria-hidden, so
            otherwise the only thing in the corner has no accessible name. */}
        <Link to="/" aria-label={t("brand")} className="flex flex-row items-center gap-2">
          {/* The sprite, not favicon.svg: an external <use> with no fragment
              renders nothing in Firefox. */}
          <svg className="xs:h-8 xs:w-6 h-6 w-5" aria-hidden="true">
            <use href="/icons.svg#brand-mark" />
          </svg>
          {/* Mobile-first: the wordmark is hidden by default and appears once
              there is room for it. */}
          <span className="text-accent hidden text-lg font-bold sm:inline">{t("brand")}</span>
        </Link>
        {/* Four icons and the menu, at every width: everything that used to
            wrap here - orders, following, profile, the three admin links -
            lives in UserMenu now. */}
        <nav className="xs:gap-4 flex items-center justify-end gap-3 text-sm">
          <LanguageSwitcher />
          <Link to="/" aria-label={t("nav.home")} className={iconLink}>
            <svg className={iconSize} aria-hidden="true">
              <use href="/icons.svg#home-icon" />
            </svg>
          </Link>
          <Link to="/search" aria-label={t("nav.search")} className={iconLink}>
            <svg className={iconSize} aria-hidden="true">
              <use href="/icons.svg#search-icon" />
            </svg>
          </Link>
          <Link to="/addlisting" aria-label={t("nav.addListing")} className={iconLink}>
            <svg className={iconSize} aria-hidden="true">
              <use href="/icons.svg#add-icon" />
            </svg>
          </Link>
          <NotificationBell />
          <UserMenu />
        </nav>
      </div>
    </header>
  );
}
