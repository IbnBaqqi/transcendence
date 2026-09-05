import { useRef, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useAuth } from "../../hooks/useAuth";
import { useDismissable } from "../../hooks/useDismissable";
import { useOwnProfile } from "../../api/profile";
import { useModal } from "../../providers/modalContext";
import { deriveInitials } from "../../lib/initials";
import Avatar from "../objects/Avatar.tsx";
import { Skeleton } from "../objects/Skeleton";

const row =
  "text-foreground hover:bg-surface-soft flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-sm";

// The whole path is the prop, not the symbol id: iconSprite.test.ts greps the
// source for /icons.svg#<id>, and a composed href is invisible to it.
function MenuIcon({ href }: { href: string }) {
  return (
    <svg className="h-5 w-4 shrink-0" aria-hidden="true">
      <use href={href} />
    </svg>
  );
}

export function UserMenu() {
  const { t } = useTranslation();
  const { user, isLoading: authLoading, logout } = useAuth();
  const navigate = useNavigate();
  const { openModal, openChat } = useModal();

  // The picture lives on the profile, not the auth session, so the menu has
  // to ask for it - disabled while signed out, where it would only 401.
  const { data: profile } = useOwnProfile({ enabled: Boolean(user) });

  const [open, setOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  useDismissable(open, setOpen, rootRef, triggerRef);

  // Neither branch until we know who this is. A menu offering "Log In" is a
  // claim about the viewer, and it is the wrong one on every reload by a
  // signed-in user.
  if (authLoading) return <Skeleton className="h-8 w-8 rounded-full" />;

  const close = () => setOpen(false);

  async function handleLogout() {
    setLoggingOut(true);
    try {
      await logout();
      close();
      // Home rather than staying put: logout reaches every page from here now,
      // including /admin/*, where RequireAdmin answers a signed-out viewer
      // with the 404.
      navigate("/");
    } finally {
      setLoggingOut(false);
    }
  }

  return (
    <div ref={rootRef} className="relative">
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-label={t("nav.menu")}
        aria-haspopup="true"
        aria-expanded={open}
        className="focus:ring-accent inline-flex cursor-pointer items-center rounded-full focus:ring-2 focus:ring-offset-2 focus:outline-none"
      >
        <Avatar
          size="sm"
          initials={user ? deriveInitials(user.username) : "?"}
          imageUrl={profile?.avatar_url ?? undefined}
          interactive
        />
      </button>

      {open && (
        // role="group", not "menu": role="menu" promises arrow-key roving
        // focus and typeahead, and these are links you Tab through.
        <div
          role="group"
          aria-label={t("nav.menu")}
          className="border-line bg-surface absolute top-full right-0 z-20 mt-2 w-56 rounded border py-1 shadow-lg"
        >
          {user ? (
            <>
              <Link to="/profile" onClick={close} className={row}>
                <MenuIcon href="/icons.svg#profile-icon" />
                {t("nav.profile")}
              </Link>
              <Link to="/dashboard" onClick={close} className={row}>
                <MenuIcon href="/icons.svg#dashboard-icon" />
                {t("pages.dashboard.title")}
              </Link>
              <Link to="/orders" onClick={close} className={row}>
                <MenuIcon href="/icons.svg#orders-icon" />
                {t("nav.orders")}
              </Link>
              <Link to="/following" onClick={close} className={row}>
                <MenuIcon href="/icons.svg#following-icon" />
                {t("nav.following")}
              </Link>
              <button
                type="button"
                onClick={() => {
                  close();
                  openChat();
                }}
                className={row}
              >
                <MenuIcon href="/icons.svg#chat-icon" />
                {t("nav.openChat")}
              </button>
              {user.role === "ADMIN" && (
                <div className="border-line mt-1 border-t pt-1">
                  <Link to="/admin/listings" onClick={close} className={row}>
                    <MenuIcon href="/icons.svg#admin-icon" />
                    {t("nav.admin")}
                  </Link>
                  <Link to="/admin/users" onClick={close} className={row}>
                    <MenuIcon href="/icons.svg#users-admin-icon" />
                    {t("nav.adminUsers")}
                  </Link>
                  <Link to="/admin/orders" onClick={close} className={row}>
                    <MenuIcon href="/icons.svg#orders-admin-icon" />
                    {t("nav.adminOrders")}
                  </Link>
                </div>
              )}
              <div className="border-line mt-1 border-t pt-1">
                <button
                  type="button"
                  disabled={loggingOut}
                  onClick={() => void handleLogout()}
                  className={`${row} disabled:cursor-not-allowed disabled:opacity-50`}
                >
                  <MenuIcon href="/icons.svg#logout-icon" />
                  {loggingOut ? t("common.loggingOut") : t("common.logOut")}
                </button>
              </div>
            </>
          ) : (
            <>
              <button
                type="button"
                onClick={() => {
                  close();
                  openModal("login");
                }}
                className={row}
              >
                {t("common.logIn")}
              </button>
              <button
                type="button"
                onClick={() => {
                  close();
                  openModal("register");
                }}
                className={row}
              >
                {t("common.register")}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
