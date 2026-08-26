package oauth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

func AssertURLs(cfg config.AuthConfig) error {
	if !cfg.Google.Configured() && !cfg.GitHub.Configured() {
		return nil
	}

	public, err := absoluteURL("PUBLIC_URL", cfg.PublicURL)
	if err != nil {
		return err
	}
	frontend, err := absoluteURL("FRONTEND_URL", cfg.FrontendURL)
	if err != nil {
		return err
	}

	if path := strings.Trim(public.Path, "/"); path != "" {
		return fmt.Errorf(
			"PUBLIC_URL must not include a path (got %q).\n"+
				"The OAuth state cookie is scoped to /api/v1/auth/oauth, so a prefix\n"+
				"would send the callback somewhere the cookie is never sent - every\n"+
				"sign-in would fail its state check with nothing in the logs.\n\n"+
				"    PUBLIC_URL=%s://%s",
			public.Path, public.Scheme, public.Host)
	}

	if !sameSite(public.Hostname(), frontend.Hostname()) {
		return fmt.Errorf(
			"PUBLIC_URL (%s) and FRONTEND_URL (%s) must share a site.\n"+
				"The session cookie is SameSite=Lax, so a frontend on an unrelated\n"+
				"host never receives it: OAuth would appear to succeed and leave the\n"+
				"user logged out.\n\n"+
				"Use the same hostname, or make one a subdomain of the other -\n"+
				"api.example.com and example.com are fine, and ports do not matter",
			public.Hostname(), frontend.Hostname())
	}

	return nil
}

func absoluteURL(name, raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s is not a valid URL (%q): %w", name, raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s must start with http:// or https:// (got %q)", name, raw)
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("%s has no host (got %q)", name, raw)
	}
	return parsed, nil
}

// Hostnames only: cookies ignore ports, which is why localhost:8080 and
// localhost:5173 are one site. A subdomain test rather than a
// registrable-domain one - it does not consult the public suffix list.
func sameSite(a, b string) bool {
	a, b = strings.ToLower(a), strings.ToLower(b)
	if a == b {
		return true
	}
	return strings.HasSuffix(a, "."+b) || strings.HasSuffix(b, "."+a)
}
