package auth

import (
	"context"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/oauth"
)

func TestAPasswordAccountSaysSoAndListsNoProviders(t *testing.T) {
	svc, _ := newService(t)

	res, err := svc.Signup(context.Background(), signupInput("forager"))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	if !res.User.HasPassword {
		t.Error("has_password = false on an account created with a password")
	}
	if len(res.User.Providers) != 0 {
		t.Errorf("providers = %q, want none", res.User.Providers)
	}
	if res.User.Providers == nil {
		t.Error("providers is nil - it marshals as null, and the frontend reads its length")
	}
}

func TestAnOAuthAccountNamesItsProviderAndHasNoPassword(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	// The identity is linked inside LoginWithIdentity's own transaction, and
	// UserInfo reads the pool - so this also pins that the read happens after
	// the commit. Move it earlier and providers comes back empty here.
	first, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test"))
	if err != nil {
		t.Fatalf("oauth sign-in: %v", err)
	}

	if first.User.HasPassword {
		t.Error("has_password = true on an account created through a provider")
	}
	if len(first.User.Providers) != 1 || first.User.Providers[0] != oauth.ProviderGoogle {
		t.Errorf("providers = %q on first sign-in, want [%s]", first.User.Providers, oauth.ProviderGoogle)
	}

	second, err := svc.LoginWithIdentity(ctx, googleLogin("g-1", "aino@example.test"))
	if err != nil {
		t.Fatalf("returning oauth sign-in: %v", err)
	}
	if len(second.User.Providers) != 1 || second.User.Providers[0] != oauth.ProviderGoogle {
		t.Errorf("providers = %q on a returning sign-in, want [%s]", second.User.Providers, oauth.ProviderGoogle)
	}
	if second.User.HasPassword {
		t.Error("has_password = true for a returning oauth user")
	}
}

func TestSignupAndRefreshDescribeTheAccountTheSameWay(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	signup, err := svc.Signup(ctx, signupInput("forager"))
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	refreshed, err := svc.RedeemSession(ctx, signup.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if refreshed.User.HasPassword != signup.User.HasPassword {
		t.Errorf("has_password = %v after refresh, was %v at signup",
			refreshed.User.HasPassword, signup.User.HasPassword)
	}
	if len(refreshed.User.Providers) != len(signup.User.Providers) {
		t.Errorf("providers = %q after refresh, was %q at signup",
			refreshed.User.Providers, signup.User.Providers)
	}
}
