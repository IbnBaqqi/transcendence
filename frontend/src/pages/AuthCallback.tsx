import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useTranslation } from "react-i18next";

import { useAuth } from "../hooks/useAuth";

// The backend never puts the access token in the redirect URL - it only sets
// the refresh cookie, then bounces us here. Success simply means "there's
// probably a cookie now, go revive the session". Errors arrive as a URL slug,
// not JSON, so axios error handling never applies to this page (see
// OAUTH_FRONTEND_HANDOFF.md for the full contract).
// Keys rather than sentences, resolved at render: the page can outlive a
// language change, and a string captured here would not follow it.
const ERROR_KEYS: Record<string, string> = {
  access_denied: "auth.callback.accessDenied",
  invalid_state: "auth.callback.invalidState",
  invalid_request: "auth.callback.invalidRequest",
  no_email: "auth.callback.noEmail",
  email_in_use: "auth.callback.emailInUse",
  already_linked: "auth.callback.alreadyLinked",
  retry: "auth.callback.retry",
  server_error: "auth.callback.serverError",
};

export default function AuthCallback() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { restoreSession } = useAuth();
  const errorSlug = searchParams.get("error");
  const message = useMemo(
    () => (errorSlug ? t(ERROR_KEYS[errorSlug] ?? ERROR_KEYS.server_error) : null),
    [errorSlug, t],
  );
  const [restoring, setRestoring] = useState(true);

  useEffect(() => {
    if (errorSlug) return;
    let cancelled = false;
    // force: the cookie is the brand-new identity, so any token left over from
    // whoever was signed in before the OAuth redirect must not win.
    restoreSession({ force: true }).then((ok) => {
      if (cancelled) return;
      if (ok) navigate("/", { replace: true });
      // No recoverable session even though the redirect claimed success - leave
      // the generic message up so the user isn't staring at a blank screen.
      else setRestoring(false);
    });
    return () => {
      cancelled = true;
    };
  }, [errorSlug, navigate, restoreSession]);

  return (
    <div className="mx-auto flex max-w-md flex-col items-center justify-center px-4 py-24 text-center">
      {message ? (
        <>
          <h1 className="text-foreground text-page-title font-bold">
            {errorSlug === "access_denied"
              ? t("auth.callback.cancelledTitle")
              : t("auth.callback.failedTitle")}
          </h1>
          <p className="text-muted mt-2">{message}</p>
          <a href="/" className="text-accent mt-6 inline-block hover:underline">
            {t("auth.callback.backHome")}
          </a>
        </>
      ) : restoring ? (
        <p className="text-muted">{t("auth.callback.completing")}</p>
      ) : (
        <>
          <h1 className="text-foreground text-page-title font-bold">
            {t("auth.callback.failedTitle")}
          </h1>
          <p className="text-muted mt-2">{t("auth.callback.noSession")}</p>
          <a href="/" className="text-accent mt-6 inline-block hover:underline">
            {t("auth.callback.backHome")}
          </a>
        </>
      )}
    </div>
  );
}
