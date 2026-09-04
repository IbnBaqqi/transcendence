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
import { Skeleton } from "../objects/Skeleton";

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  isActive ? "text-foreground" : "text-muted hover:text-foreground";

export default function Header() {
  const { user, isLoading: authLoading } = useAuth();
  const { openModal, openChat } = useModal();
  const { t } = useTranslation();

  // The picture lives on the profile, not the auth session, so the header has
  // to ask for it - disabled while signed out, where it would only 401.
  const { data: profile } = useOwnProfile({ enabled: Boolean(user) });

  const initials = user ? deriveInitials(user.username) : "?";

  return (
    <header className="border-line bg-surface sticky top-0 z-10 border-b">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
        <div className="flex flex-row items-center gap-2">
          <Link to="/">
            <svg className="h-8 w-6" aria-hidden="true">
              <use href="/favicon.svg" />
            </svg>
          </Link>
          <Link to="/" className="text-accent text-lg font-bold">
            {t("brand")}
          </Link>
        </div>
        <nav className="flex items-center gap-4 text-sm">
          <LanguageSwitcher />
          {/* {navLinkClass} passes the function itself and React Router calls it and supplies { isActive } */}
          {/* <NavLink to="/" className={navLinkClass}> */}
          {/*   {t("nav.home")} */}
          {/* </NavLink> */}
          <Link to="/" aria-label={t("nav.home")} className="text-muted hover:text-foreground">
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#home-icon" />
            </svg>
          </Link>
          <Link
            to="/search"
            aria-label={t("nav.search")}
            className="text-muted hover:text-foreground"
          >
            <svg className="h-6 w-5" aria-hidden="true">
              <use href="/icons.svg#search-icon" />
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
          {authLoading ? (
            // Neither branch until we know who this is. The log-in button is a
            // claim about the viewer, and it is the wrong one on every reload
            // by a signed-in user.
            <Skeleton className="h-8 w-8 rounded-full" />
          ) : user ? (
            <>
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
                to="/orders"
                aria-label={t("nav.orders")}
                className="text-muted hover:text-foreground"
              >
                <svg className="h-6 w-5" aria-hidden="true">
                  <use href="/icons.svg#orders-icon" />
                </svg>
              </Link>
              <Link
                to="/following"
                aria-label={t("nav.following")}
                className="text-muted hover:text-foreground"
              >
                <svg className="h-6 w-5" aria-hidden="true">
                  <use href="/icons.svg#following-icon" />
                </svg>
              </Link>
              <Link to="/profile" aria-label={t("nav.profile")}>
                <Avatar
                  size="sm"
                  initials={initials}
                  imageUrl={profile?.avatar_url ?? undefined}
                  interactive
                />
              </Link>
            </>
          ) : (
            <button type="button" aria-label={t("common.logIn")} onClick={() => openModal("login")}>
              <Avatar size="sm" initials="?" interactive />
            </button>
          )}
          {user?.role === "ADMIN" && (
            <>
              <NavLink to="/admin/listings" className={navLinkClass}>
                {t("nav.admin")}
              </NavLink>
              <NavLink to="/admin/users" className={navLinkClass}>
                {t("nav.adminUsers")}
              </NavLink>
              <NavLink to="/admin/orders" className={navLinkClass}>
                {t("nav.adminOrders")}
              </NavLink>
            </>
          )}
          {/* TODO: add Listing (#20) and auth links (#46) when those pages exist */}
        </nav>
      </div>
    </header>
  );
}
