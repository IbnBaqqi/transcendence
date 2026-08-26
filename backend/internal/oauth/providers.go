package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

const (
	ProviderGoogle = "google"
	ProviderGitHub = "github"
)

const maxUserInfoBytes = 1 << 20 // 1 MiB

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxUserInfoBytes)).Decode(dst); err != nil {
		return fmt.Errorf("GET %s: decode response: %w", url, err)
	}
	return nil
}

func newGoogle(cfg config.OAuthConfig, publicURL string) *Provider {
	return &Provider{
		Name: ProviderGoogle,
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  RedirectURI(publicURL, ProviderGoogle),
			Scopes:       []string{"openid", "email"},
			// #nosec G101 -- published endpoint URLs, not credentials
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
		fetch: googleIdentity,
	}
}

type googleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func googleIdentity(ctx context.Context, client *http.Client) (Identity, error) {
	var u googleUser
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &u); err != nil {
		return Identity{}, err
	}

	identity := Identity{ProviderUserID: u.Sub}
	if u.EmailVerified {
		identity.Email = u.Email
	}
	return identity, nil
}

func newGitHub(cfg config.OAuthConfig, publicURL string) *Provider {
	return &Provider{
		Name: ProviderGitHub,
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  RedirectURI(publicURL, ProviderGitHub),
			Scopes:       []string{"user:email"},
			// #nosec G101 -- published endpoint URLs, not credentials
			Endpoint: oauth2.Endpoint{
				AuthURL:   "https://github.com/login/oauth/authorize",
				TokenURL:  "https://github.com/login/oauth/access_token",
				AuthStyle: oauth2.AuthStyleInParams,
			},
		},
		fetch: githubIdentity,
	}
}

type githubUser struct {
	ID int64 `json:"id"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func githubIdentity(ctx context.Context, client *http.Client) (Identity, error) {
	var u githubUser
	if err := getJSON(ctx, client, "https://api.github.com/user", &u); err != nil {
		return Identity{}, err
	}

	identity := Identity{ProviderUserID: strconv.FormatInt(u.ID, 10)}

	var emails []githubEmail
	if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
		return Identity{}, err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			identity.Email = e.Email
			break
		}
	}

	return identity, nil
}
