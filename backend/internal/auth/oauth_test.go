package auth

import "testing"

func TestUsernameFromEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{"plain", "forager@example.test", "forager"},
		{"lowercased", "Sam.Forager@example.test", "sam.forager"},
		{"plus tag dropped", "sam+berries@example.test", "sam"},
		{"unsupported runes dropped", "sam'o~brien@example.test", "samobrien"},
		{"trimmed of separators", "...sam...@example.test", "sam"},
		{"no usable characters", "日本語@example.test", "forager"},
		{"empty local part", "@example.test", "forager"},
		{
			"truncated to the limit",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa@example.test",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		{
			"truncation does not leave a trailing separator",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbb@example.test",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := usernameFromEmail(tc.email)
			if got != tc.want {
				t.Errorf("usernameFromEmail(%q) = %q, want %q", tc.email, got, tc.want)
			}
			if len(got) > maxDerivedUsername {
				t.Errorf("username %q is longer than the %d limit", got, maxDerivedUsername)
			}
		})
	}
}

func TestProviderLabel(t *testing.T) {
	if got := providerLabel("google"); got != "Google" {
		t.Errorf("google label = %q, want Google", got)
	}
	if got := providerLabel("github"); got != "GitHub" {
		t.Errorf("github label = %q, want GitHub", got)
	}
	if got := providerLabel("gitlab"); got != "gitlab" {
		t.Errorf("unknown label = %q, want it passed through", got)
	}
}

func TestRandomSuffixIsNotConstant(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		s := randomSuffix()
		if len(s) != 8 {
			t.Fatalf("suffix %q is %d characters, want 8", s, len(s))
		}
		seen[s] = true
	}
	if len(seen) < 90 {
		t.Errorf("only %d distinct suffixes in 100 draws - not random enough to break a tie", len(seen))
	}
}
