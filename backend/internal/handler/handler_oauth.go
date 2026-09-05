package handler

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/IbnBaqqi/transcendence/internal/auth"
)

const (
	oauthStatePrefix  = "oauth_state_"
	oauthStatePath    = "/api/v1/auth/oauth"
	oauthStateTTL     = 10 * time.Minute
	oauthFrontendPath = "/auth/callback"
)

func oauthStateCookie(provider string) string {
	return oauthStatePrefix + provider
}

// The registry only holds a provider whose credentials are set, so this is
// what the sign-in buttons render from. Without it the frontend advertises
// every provider the code knows about and a misconfigured one sends the user
// to a JSON error page, outside the app.
func (h *Handler) OAuthProviders(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, h.oauth.Names())
}

func (h *Handler) OAuthStart(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "provider")

	provider, ok := h.oauth.Get(name)
	if !ok {
		respondWithError(w, http.StatusNotFound, "Unknown sign-in provider")
		return
	}

	state, err := randomState()
	if err != nil {
		slog.Error("oauth: generating state failed",
			"request_id", middleware.GetReqID(r.Context()), "error", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	h.setOAuthStateCookie(w, name, state, oauthStateTTL)

	// #nosec G710 -- the URL is built from the provider's configured endpoint,
	// never from the request; only the registry decides which provider applies
	http.Redirect(w, r, provider.AuthCodeURL(state), http.StatusFound)
}

func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "provider")

	provider, ok := h.oauth.Get(name)
	if !ok {
		respondWithError(w, http.StatusNotFound, "Unknown sign-in provider")
		return
	}

	w.Header().Set("Referrer-Policy", "no-referrer")

	h.setOAuthStateCookie(w, name, "", -1)

	query := r.URL.Query()

	if query.Get("error") != "" {
		h.redirectToFrontend(w, r, "access_denied")
		return
	}

	cookie, err := r.Cookie(oauthStateCookie(name))
	if err != nil || !sameState(cookie.Value, query.Get("state")) {
		h.redirectToFrontend(w, r, "invalid_state")
		return
	}

	code := query.Get("code")
	if code == "" {
		h.redirectToFrontend(w, r, "invalid_request")
		return
	}

	identity, err := provider.Exchange(r.Context(), code)
	if err != nil {
		slog.Error("oauth: code exchange failed", "provider", name,
			"request_id", middleware.GetReqID(r.Context()), "error", err)
		h.redirectToFrontend(w, r, "server_error")
		return
	}

	result, err := h.Auth.LoginWithIdentity(r.Context(), auth.OAuthLogin{
		Provider:       name,
		ProviderUserID: identity.ProviderUserID,
		Email:          identity.Email,
	})
	if err != nil {
		h.oauthFailure(w, r, err)
		return
	}

	h.setRefreshTokenCookie(w, result.RefreshToken, auth.RefreshTokenTTL)
	h.redirectToFrontend(w, r, "")
}

func (h *Handler) oauthFailure(w http.ResponseWriter, r *http.Request, err error) {
	var validation *auth.ValidationError
	var conflict *auth.ConflictError
	var accountExists *auth.AccountExistsError
	var retry *auth.RetryError

	switch {
	case errors.As(err, &validation):
		h.redirectToFrontend(w, r, "no_email")
	case errors.As(err, &accountExists):
		h.redirectToFrontend(w, r, "email_in_use")
	case errors.As(err, &retry):
		h.redirectToFrontend(w, r, "retry")
	case errors.As(err, &conflict):
		h.redirectToFrontend(w, r, "already_linked")
	default:
		slog.Error("oauth: sign-in failed",
			"request_id", middleware.GetReqID(r.Context()), "error", err)
		h.redirectToFrontend(w, r, "server_error")
	}
}

func (h *Handler) redirectToFrontend(w http.ResponseWriter, r *http.Request, slug string) {
	target, err := url.Parse(h.frontendURL)
	if err != nil {
		slog.Error("oauth: FRONTEND_URL is not a valid URL",
			"value", h.frontendURL, "error", err)
		respondWithError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}

	target.Path = strings.TrimSuffix(target.Path, "/") + oauthFrontendPath

	q := target.Query()
	if slug == "" {
		q.Set("status", "ok")
	} else {
		q.Set("error", slug)
	}
	target.RawQuery = q.Encode()

	http.Redirect(w, r, target.String(), http.StatusFound)
}

func (h *Handler) setOAuthStateCookie(w http.ResponseWriter, provider, value string, ttl time.Duration) {
	maxAge := int(ttl.Seconds())
	if ttl < 0 {
		maxAge = -1
	}
	// #nosec G124 -- Secure follows COOKIE_SECURE, true by default
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie(provider),
		Value:    value,
		Path:     oauthStatePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		// Lax is required: the callback is a top-level cross-site navigation,
		// which Strict would withhold the cookie on.
		SameSite: http.SameSiteLaxMode,
	})
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func sameState(cookie, query string) bool {
	if cookie == "" || query == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie), []byte(query)) == 1
}
