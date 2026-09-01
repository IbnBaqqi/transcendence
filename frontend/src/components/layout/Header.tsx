// Link: normal navigation link (no "am I active?" info)
// NavLink: a Link that knows if it points to the current page, so you can style the active one differently
import { useModal } from "../../providers/modalContext";
import { useAuth } from "../../hooks/useAuth";
import { deriveInitials } from "../../lib/initials";
import { Link, NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import Avatar from "../objects/Avatar.tsx";
import { LanguageSwitcher } from "./LanguageSwitcher";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  isActive ? "text-foreground" : "text-muted hover:text-foreground";

export default function Header() {
  const { user } = useAuth();
  const { openModal, openChat } = useModal();
  const { t } = useTranslation();

  // Names live on the profile, not the auth session - header works from the
  // username alone and "?" covers signed-out visitors.
  const initials = user ? deriveInitials(user.username) : "?";

  return (
    <header className="border-line bg-surface sticky top-0 z-10 border-b">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <Link to="/" className="text-accent text-lg font-bold">
          {t("brand")}
        </Link>
        <nav className="flex items-center gap-6 text-sm">
          <LanguageSwitcher />
          {/* {navLinkClass} passes the function itself and React Router calls it and supplies { isActive } */}
          <NavLink to="/" className={navLinkClass}>
            {t("nav.home")}
          </NavLink>
          <button
            type="button"
            onClick={openChat}
            aria-label={t("nav.openChat")}
            className="text-muted hover:text-foreground"
          >
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#chat-icon" />
            </svg>
          </button>
          <Link
            to="/notifications"
            aria-label={t("nav.notifications")}
            className="text-muted hover:text-foreground"
          >
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#notifications-icon" />
            </svg>
          </Link>
          <Link
            to="/addlisting"
            aria-label={t("nav.addListing")}
            className="text-muted hover:text-foreground"
          >
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#add-icon" />
            </svg>
          </Link>
          {user ? (
            <Link to="/profile">
              <Avatar size="sm" initials={initials} interactive />
            </Link>
          ) : (
            <button type="button" onClick={() => openModal("login")}>
              <Avatar size="sm" initials="?" interactive />
            </button>
          )}
          {/* TODO: add Listing (#20) and auth links (#46) when those pages exist */}
        </nav>
      </div>
    </header>
  );
}
