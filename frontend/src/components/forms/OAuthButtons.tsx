import { useTranslation } from "react-i18next";

import { API_BASE_URL } from "../../api/client";

// "Sign in with Google / GitHub" buttons. These MUST be plain <a> tags doing a
// top-level navigation (not buttons firing axios) - the OAuth start route is a
// 302 redirect to the provider's consent screen, and the response is a whole
// new page, not JSON.
//
// The href is built from API_BASE_URL (the same base the API client uses), so
// it respects VITE_API_URL and stays in step with every other request.
function oauthStartHref(provider: string): string {
  // API_BASE_URL always carries the /api/v1 segment (see frontend/.env.example);
  // strip a possible trailing slash so the concatenation is clean.
  return `${API_BASE_URL.replace(/\/$/, "")}/auth/oauth/${provider}`;
}

export default function OAuthButtons() {
  const { t } = useTranslation();
  const base =
    "flex w-full items-center justify-center gap-2 rounded-full border border-line px-4 py-2 " +
    "font-medium transition-colors duration-150 hover:bg-soft-hover active:bg-soft-active " +
    "focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2";

  return (
    <div className="space-y-2">
      <a href={oauthStartHref("google")} className={base}>
        <svg className="h-5 w-5" aria-hidden="true" viewBox="0 0 24 24">
          <path
            fill="#4285F4"
            d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.27-4.74 3.27-8.1Z"
          />
          <path
            fill="#34A853"
            d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23Z"
          />
          <path
            fill="#FBBC05"
            d="M5.84 14.1a6.6 6.6 0 0 1 0-4.2V7.06H2.18a11 11 0 0 0 0 9.88l3.66-2.84Z"
          />
          <path
            fill="#EA4335"
            d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.06l3.66 2.84c.87-2.6 3.3-4.52 6.16-4.52Z"
          />
        </svg>
        {t("auth.continueWithGoogle")}
      </a>
      <a href={oauthStartHref("github")} className={base}>
        <svg className="h-5 w-5" aria-hidden="true" viewBox="0 0 16 16" fill="currentColor">
          <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82a7.42 7.42 0 0 1 4 0c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
        </svg>
        {t("auth.continueWithGitHub")}
      </a>
    </div>
  );
}
