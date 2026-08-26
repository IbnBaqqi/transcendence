package handler

import "testing"

func TestSameState(t *testing.T) {
	const state = "8f14e45fceea167a5a36dedd4bea2543"

	tests := []struct {
		name   string
		cookie string
		query  string
		want   bool
	}{
		{"match", state, state, true},
		{"mismatch", state, "00000000000000000000000000000000", false},
		{"no cookie", "", state, false},
		{"no query value", state, "", false},
		{"both empty", "", "", false},
		{"query is a prefix of the cookie", state, state[:16], false},
		{"cookie is a prefix of the query", state[:16], state, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sameState(tc.cookie, tc.query); got != tc.want {
				t.Errorf("sameState(%q, %q) = %v, want %v", tc.cookie, tc.query, got, tc.want)
			}
		})
	}
}

func TestOAuthStateCookieIsPerProvider(t *testing.T) {
	if oauthStateCookie("google") == oauthStateCookie("github") {
		t.Error("both providers share a cookie name - a state issued for one could be replayed against the other")
	}
}

func TestRandomStateIsNotConstant(t *testing.T) {
	first, err := randomState()
	if err != nil {
		t.Fatalf("randomState: %v", err)
	}
	if len(first) != 64 {
		t.Errorf("state is %d characters, want 64 (32 bytes hex)", len(first))
	}

	second, err := randomState()
	if err != nil {
		t.Fatalf("randomState: %v", err)
	}
	if first == second {
		t.Error("two states came back identical")
	}
}
