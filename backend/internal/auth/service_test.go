package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func TestValidateSignupInput(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*dtos.CreateUserRequest)
		wantErr bool
	}{
		{"a normal signup", func(*dtos.CreateUserRequest) {}, false},
		{"no username", func(i *dtos.CreateUserRequest) { i.Username = "" }, true},
		{"username at the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("a", 50) }, false},
		{"username over the limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("a", 51) }, true},
		{"multi-byte username over the BYTE limit", func(i *dtos.CreateUserRequest) { i.Username = strings.Repeat("ä", 26) }, true},
		{"no email", func(i *dtos.CreateUserRequest) { i.Email = "" }, true},
		{"password too short", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 7) }, true},
		{"password at the floor", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 8) }, false},
		{"password at the ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 72) }, false},
		{"password over the ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("a", 73) }, true},
		{"multi-byte password over the BYTE ceiling", func(i *dtos.CreateUserRequest) { i.Password = strings.Repeat("ä", 37) }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := signupInput("valid")
			tt.mutate(&input)

			err := validateSignupInput(input)
			if tt.wantErr {
				var v *ValidationError
				if !errors.As(err, &v) {
					t.Fatalf("err = %v, want *ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
