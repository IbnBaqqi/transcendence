import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";

// The backend never puts the access token in the redirect URL - it only sets
// the refresh cookie, then bounces us here. Success simply means "there's
// probably a cookie now, go revive the session". Errors arrive as a URL slug,
// not JSON, so axios error handling never applies to this page (see
// OAUTH_FRONTEND_HANDOFF.md for the full contract).
const ERROR_MESSAGES: Record<string, string> = {
  access_denied: "Sign-in was cancelled. No changes were made.",
  invalid_state: "Something went wrong with the sign-in. Please try again.",
  invalid_request: "Something went wrong with the sign-in. Please try again.",
  no_email: "Your provider account doesn't have a verified email.",
  email_in_use: "An account with this email already exists. Sign in with your password instead.",
  already_linked: "This account is already linked to another user.",
  retry: "Something went wrong. Please try again.",
  server_error: "Something went wrong on our end. Please try again.",
};

export default function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { restoreSession } = useAuth();
  const errorSlug = searchParams.get("error");
  const message = useMemo(
    () => (errorSlug ? (ERROR_MESSAGES[errorSlug] ?? ERROR_MESSAGES.server_error) : null),
    [errorSlug],
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
          <h1 className="text-foreground text-2xl font-bold">
            {errorSlug === "access_denied" ? "Sign-in cancelled" : "Sign-in didn't complete"}
          </h1>
          <p className="text-muted mt-2">{message}</p>
          <a href="/" className="text-accent mt-6 inline-block hover:underline">
            Back to home
          </a>
        </>
      ) : restoring ? (
        <p className="text-muted">Completing sign-in…</p>
      ) : (
        <>
          <h1 className="text-foreground text-2xl font-bold">Sign-in didn't complete</h1>
          <p className="text-muted mt-2">We couldn't start a session.</p>
          <a href="/" className="text-accent mt-6 inline-block hover:underline">
            Back to home
          </a>
        </>
      )}
    </div>
  );
}
