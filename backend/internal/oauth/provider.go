package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

const callbackTimeout = 10 * time.Second

type Identity struct {
	ProviderUserID string
	Email          string
}

type fetchIdentity func(ctx context.Context, client *http.Client) (Identity, error)

type Provider struct {
	Name  string
	cfg   *oauth2.Config
	fetch fetchIdentity
}

type Registry struct {
	providers map[string]*Provider
}

func NewRegistry(cfg config.AuthConfig) *Registry {
	r := &Registry{providers: make(map[string]*Provider)}

	if cfg.Google.Configured() {
		r.add(newGoogle(cfg.Google, cfg.PublicURL))
	}
	if cfg.GitHub.Configured() {
		r.add(newGitHub(cfg.GitHub, cfg.PublicURL))
	}

	return r
}

func (r *Registry) add(p *Provider) {
	r.providers[p.Name] = p
}

func (r *Registry) Get(name string) (*Provider, bool) {
	if r == nil {
		return nil, false
	}
	p, ok := r.providers[name]
	return p, ok
}

func RedirectURI(publicURL, provider string) string {
	return strings.TrimSuffix(publicURL, "/") + "/api/v1/auth/oauth/" + provider + "/callback"
}

func (p *Provider) AuthCodeURL(state string) string {
	return p.cfg.AuthCodeURL(state)
}

func (p *Provider) Exchange(ctx context.Context, code string) (Identity, error) {
	ctx, cancel := context.WithTimeout(ctx, callbackTimeout)
	defer cancel()

	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: callbackTimeout})

	token, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return Identity{}, fmt.Errorf("oauth %s: exchange code: %w", p.Name, err)
	}

	identity, err := p.fetch(ctx, p.cfg.Client(ctx, token))
	if err != nil {
		return Identity{}, fmt.Errorf("oauth %s: fetch identity: %w", p.Name, err)
	}

	if identity.ProviderUserID == "" {
		return Identity{}, fmt.Errorf("oauth %s: provider returned no account id", p.Name)
	}

	return identity, nil
}
