package oauth

import (
	"encoding/json"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/config"
)

func configured() config.OAuthConfig {
	return config.OAuthConfig{ClientID: "id", ClientSecret: "secret"}
}

func TestNamesListsOnlyConfiguredProviders(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.AuthConfig
		want []string
	}{
		{"neither", config.AuthConfig{}, []string{}},
		{"google only", config.AuthConfig{Google: configured()}, []string{"google"}},
		{"github only", config.AuthConfig{GitHub: configured()}, []string{"github"}},
		{
			"both, alphabetical",
			config.AuthConfig{Google: configured(), GitHub: configured()},
			[]string{"github", "google"},
		},
		{
			"an id without a secret is not configured",
			config.AuthConfig{Google: config.OAuthConfig{ClientID: "id"}},
			[]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewRegistry(tc.cfg).Names()
			if len(got) != len(tc.want) {
				t.Fatalf("Names() = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("Names()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// Map iteration is randomised, so an unsorted Names() passes a single run and
// reorders the sign-in buttons between requests. Repeat until that would show.
func TestNamesIsStableAcrossCalls(t *testing.T) {
	r := NewRegistry(config.AuthConfig{Google: configured(), GitHub: configured()})

	first := r.Names()
	for i := 0; i < 50; i++ {
		got := r.Names()
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("call %d returned %v, first call returned %v", i, got, first)
			}
		}
	}
}

// A nil slice marshals to `null`, which the frontend cannot map over. The
// empty case is the one that would ship it.
func TestNamesMarshalsToAnArrayWhenEmpty(t *testing.T) {
	out, err := json.Marshal(NewRegistry(config.AuthConfig{}).Names())
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if string(out) != "[]" {
		t.Errorf("marshalled to %s, want []", out)
	}
}

func TestNamesOnANilRegistry(t *testing.T) {
	var r *Registry
	if got := r.Names(); len(got) != 0 {
		t.Errorf("Names() on a nil registry = %v, want empty", got)
	}
}
