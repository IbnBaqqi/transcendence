package service

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func ptr(s string) *string { return &s }

func TestMergeString(t *testing.T) {
	current := sql.NullString{String: "before", Valid: true}

	tests := []struct {
		name string
		in   *string
		want sql.NullString
	}{
		{"not sent keeps the current value", nil, current},
		{"a value replaces", ptr("after"), sql.NullString{String: "after", Valid: true}},
		{"empty clears to NULL", ptr(""), sql.NullString{}},
		{"whitespace only clears to NULL", ptr("   "), sql.NullString{}},
		{"surrounding whitespace is trimmed", ptr("  after  "), sql.NullString{String: "after", Valid: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeString(current, tt.in)
			if got != tt.want {
				t.Errorf("mergeString = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMergeDate(t *testing.T) {
	current := sql.NullTime{Time: time.Date(1999, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true}
	future := time.Now().AddDate(1, 0, 0).Format(dtos.DateLayout)

	tests := []struct {
		name    string
		in      *string
		want    sql.NullTime
		wantErr bool
	}{
		{"not sent keeps the current value", nil, current, false},
		{"empty clears to NULL", ptr(""), sql.NullTime{}, false},
		{"a valid date parses", ptr("2001-05-14"),
			sql.NullTime{Time: time.Date(2001, 5, 14, 0, 0, 0, 0, time.UTC), Valid: true}, false},
		{"a timestamp is not a date", ptr("2001-05-14T00:00:00Z"), sql.NullTime{}, true},
		{"nonsense is rejected", ptr("yesterday"), sql.NullTime{}, true},
		{"the future is rejected", ptr(future), sql.NullTime{}, true},
		{"implausibly early is rejected", ptr("1642-12-25"), sql.NullTime{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeDate(current, tt.in)

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
			if !got.Time.Equal(tt.want.Time) || got.Valid != tt.want.Valid {
				t.Errorf("mergeDate = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestValidateProfileInput(t *testing.T) {
	tests := []struct {
		name    string
		input   dtos.UpdateProfileInput
		wantErr string
	}{
		{"an empty patch is valid", dtos.UpdateProfileInput{}, ""},
		{"firstname at the limit", dtos.UpdateProfileInput{Firstname: ptr(strings.Repeat("a", 150))}, ""},
		{"firstname over the limit", dtos.UpdateProfileInput{Firstname: ptr(strings.Repeat("a", 151))}, "firstname is too long"},
		{"lastname over the limit", dtos.UpdateProfileInput{Lastname: ptr(strings.Repeat("a", 151))}, "lastname is too long"},
		{"bio over the limit", dtos.UpdateProfileInput{Bio: ptr(strings.Repeat("a", 1001))}, "bio is too long"},
		{"phone over the limit", dtos.UpdateProfileInput{PhoneNumber: ptr(strings.Repeat("1", 16))}, "phone_number is too long"},
		{"location over the limit", dtos.UpdateProfileInput{Location: ptr(strings.Repeat("a", 101))}, "location is too long"},
		{"multi-byte location over the BYTE limit", dtos.UpdateProfileInput{Location: ptr(strings.Repeat("ä", 51))}, "location is too long"},
		{"a null byte in bio", dtos.UpdateProfileInput{Bio: ptr("a\x00b")}, "bio must be valid UTF-8 without null bytes"},
		{"a null byte in location", dtos.UpdateProfileInput{Location: ptr("Espoo\x00")}, "location must be valid UTF-8 without null bytes"},
		{"padding does not count", dtos.UpdateProfileInput{Firstname: ptr("  " + strings.Repeat("a", 150) + "  ")}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfileInput(tt.input)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			var v *ValidationError
			if !errors.As(err, &v) {
				t.Fatalf("err = %v, want *ValidationError", err)
			}
			if v.Message != tt.wantErr {
				t.Errorf("message = %q, want %q", v.Message, tt.wantErr)
			}
		})
	}
}
