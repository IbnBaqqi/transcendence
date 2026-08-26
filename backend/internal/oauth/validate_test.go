package oauth

import (
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

func withOAuth(public, frontend string) config.AuthConfig {
	return config.AuthConfig{
		Google:      config.OAuthConfig{ClientID: "id", ClientSecret: "secret"},
		PublicURL:   public,
		FrontendURL: frontend,
	}
}

func TestAssertURLsAcceptsWorkingConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		public   string
		frontend string
	}{
		{"the compose defaults", "http://localhost:8080", "http://localhost:5173"},
		{"same host, no ports", "https://example.com", "https://example.com"},
		{"api on a subdomain", "https://api.example.com", "https://example.com"},
		{"frontend on a subdomain", "https://example.com", "https://www.example.com"},
		{"trailing slash on public", "http://localhost:8080/", "http://localhost:5173"},
		{"case differs", "https://API.Example.com", "https://example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AssertURLs(withOAuth(tc.public, tc.frontend)); err != nil {
				t.Errorf("refused a working configuration: %v", err)
			}
		})
	}
}

func TestAssertURLsRefusesBrokenConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		public   string
		frontend string
		wants    string
	}{
		{"unrelated hosts", "https://api.example.com", "https://example.net", "share a site"},
		{"lookalike host", "https://example.com", "https://notexample.com", "share a site"},
		{"path on public", "https://example.com/api", "https://example.com", "must not include a path"},
		{"no scheme", "example.com", "https://example.com", "http:// or https://"},
		{"wrong scheme", "ftp://example.com", "https://example.com", "http:// or https://"},
		{"no host", "https://", "https://example.com", "no host"},
		{"empty frontend", "https://example.com", "", "FRONTEND_URL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := AssertURLs(withOAuth(tc.public, tc.frontend))
			if err == nil {
				t.Fatal("accepted a configuration that cannot work")
			}
			if !strings.Contains(err.Error(), tc.wants) {
				t.Errorf("error does not mention %q:\n%v", tc.wants, err)
			}
		})
	}
}

// Refusing to boot over settings nothing reads would be its own bug.
func TestAssertURLsIgnoresEverythingWhenNoProviderIsConfigured(t *testing.T) {
	cfg := config.AuthConfig{
		PublicURL:   "nonsense",
		FrontendURL: "also nonsense",
	}

	if err := AssertURLs(cfg); err != nil {
		t.Errorf("refused to start without any provider configured: %v", err)
	}
}
