import { Outlet } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useAuth } from "../../hooks/useAuth";
import NotFound from "../../pages/NotFound";

// Renders the 404 rather than "you are not an admin": telling a stranger the
// route exists is the one thing this can leak. Every endpoint behind it is
// guarded by RequireRole(ADMIN) server-side, so this is UX, not security.
export function RequireAdmin() {
  const { user, isLoading } = useAuth();
  const { t } = useTranslation();

  // Neither branch while the session is still being restored. Falling through
  // to the 404 here would throw a reloading admin out of their own page.
  if (isLoading) return <p className="text-muted p-8 text-sm">{t("common.loading")}</p>;

  if (user?.role !== "ADMIN") return <NotFound />;

  return <Outlet />;
}
