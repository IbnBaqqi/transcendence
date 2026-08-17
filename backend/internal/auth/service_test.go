package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func signupInput(name string) dtos.CreateUserRequest {
	return dtos.CreateUserRequest{
		Username: name,
		Email:    name + "@example.test",
		Password: "password123",
	}
}

func TestValidateSignupInput(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*dtos.CreateUserRequest)
		wantMsg string
	}{
		{"a normal signup", func(*dtos.CreateUserRequest) {}, ""},

		{"no username", func(i *dtos.CreateUserRequest) { i.Username = "" }, "username is required"},
		{"username at the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("a", 50) }, ""},
		{"username over the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("a", 51) }, "username must be 50 characters or fewer"},
		{"multi-byte username at the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("ä", 50) }, ""},
		{"multi-byte username over the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("ä", 51) }, "username must be 50 characters or fewer"},

		{"no email", func(i *dtos.CreateUserRequest) { i.Email = "" }, "email is required"},
		{"email at the limit", func(i *dtos.CreateUserRequest) { i.Email = strings.Repeat("a", 150) }, ""},
		{"email over the limit", func(i *dtos.CreateUserRequest) { i.Email = strings.Repeat("a", 151) }, "email must be 150 characters or fewer"},
		{"multi-byte email under the character limit", func(i *dtos.CreateUserRequest) { i.Email = strings.Repeat("ä", 100) }, ""},

		{"password too short", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 7) }, "password must be at least 8 bytes"},
		{"password at the floor", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 8) }, ""},
		{"password at the ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 72) }, ""},
		{"password over the ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 73) }, "password must be 72 bytes or fewer"},
		{"multi-byte password over the BYTE ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("ä", 37) }, "password must be 72 bytes or fewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := signupInput("valid")
			tt.mutate(&input)

			err := validateSignupInput(input)

			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			var v *ValidationError
			if !errors.As(err, &v) {
				t.Fatalf("err = %v, want *ValidationError", err)
			}
			if v.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", v.Message, tt.wantMsg)
			}
		})
	}
}

func TestDuplicateUserError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"username taken", &pq.Error{Code: "23505", Constraint: "users_username_uq"}, "username already taken"},
		{"email taken", &pq.Error{Code: "23505", Constraint: "users_email_uq"}, "email already in use"},
		{"some other unique index", &pq.Error{Code: "23505", Constraint: "listings_pkey"}, ""},
		{"a different pq error", &pq.Error{Code: "22001"}, ""},
		{"not a pq error at all", errors.New("boom"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := duplicateUserError(tt.err)

			if tt.wantMsg == "" {
				if got != nil {
					t.Fatalf("got %v, want nil - this would be misreported as a conflict", got)
				}
				return
			}

			var conflict *ConflictError
			if !errors.As(got, &conflict) {
				t.Fatalf("got %v, want *ConflictError", got)
			}
			if conflict.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", conflict.Message, tt.wantMsg)
			}
		})
	}
}
