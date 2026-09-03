// Link: normal navigation link (no "am I active?" info)
// NavLink: a Link that knows if it points to the current page, so you can style the active one differently
import { useModal } from "../../providers/modalContext";
import { useAuth } from "../../hooks/useAuth";
import { useOwnProfile } from "../../api/profile";
import { deriveInitials } from "../../lib/initials";
import { Link, NavLink } from "react-router-dom";
import { useTranslation } from "react-i18next";
import Avatar from "../objects/Avatar.tsx";
import { LanguageSwitcher } from "../objects/LanguageSwitcher";
import { NotificationBell } from "../objects/NotificationBell";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  isActive ? "text-foreground" : "text-muted hover:text-foreground";

export default function Header() {
  const { user } = useAuth();
  const { openModal, openChat } = useModal();
  const { t } = useTranslation();

  // The picture lives on the profile, not the auth session, so the header has
  // to ask for it - disabled while signed out, where it would only 401.
  const { data: profile } = useOwnProfile({ enabled: Boolean(user) });

  const initials = user ? deriveInitials(user.username) : "?";

  return (
    <header className="border-line bg-surface sticky top-0 z-10 border-b">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <Link to="/" className="text-accent text-lg font-bold">
          {t("brand")}
        </Link>
        <nav className="flex items-center gap-4 text-sm">
          <LanguageSwitcher />
          {/* {navLinkClass} passes the function itself and React Router calls it and supplies { isActive } */}
          <NavLink to="/" className={navLinkClass}>
            {t("nav.home")}
          </NavLink>
          <NavLink to="/search" className={navLinkClass}>
            {t("nav.search")}
          </NavLink>
          {user && (
            <NavLink to="/orders" className={navLinkClass}>
              {t("nav.orders")}
            </NavLink>
          )}
          {user && (
            <NavLink to="/following" className={navLinkClass}>
              {t("nav.following")}
            </NavLink>
          )}
          {/* openChat is wrapped, not passed straight to onClick: it takes an
              optional conversation id and would receive the click event. */}
          <button
            type="button"
            onClick={() => openChat()}
            aria-label={t("nav.openChat")}
            className="text-muted hover:text-foreground"
          >
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#chat-icon" />
            </svg>
          </button>
          <NotificationBell />
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
            <Link to="/profile" aria-label={t("nav.profile")}>
              <Avatar
                size="sm"
                initials={initials}
                imageUrl={profile?.avatar_url ?? undefined}
                interactive
              />
            </Link>
          ) : (
            <button type="button" aria-label={t("common.logIn")} onClick={() => openModal("login")}>
              <Avatar size="sm" initials="?" interactive />
            </button>
          )}
          {/* TODO: add Listing (#20) and auth links (#46) when those pages exist */}
        </nav>
      </div>
    </header>
  );
}
