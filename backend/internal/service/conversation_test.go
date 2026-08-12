package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

const (
	errNone       = ""
	errNotFound   = "notfound"
	errForbidden  = "forbidden"
	errConflict   = "conflict"
	errValidation = "validation"
)

func assertErrorKind(t *testing.T, err error, want string) {
	t.Helper()

	switch want {
	case errNone:
		if err != nil {
			t.Fatalf("unexpected error = %v", err)
		}
	case errNotFound:
		var target *NotFoundError
		if !errors.As(err, &target) {
			t.Fatalf("error = %v, want *NotFoundError", err)
		}
	case errForbidden:
		var target *ForbiddenError
		if !errors.As(err, &target) {
			t.Fatalf("error = %v, want *ForbiddenError", err)
		}
	case errConflict:
		var target *ConflictError
		if !errors.As(err, &target) {
			t.Fatalf("error = %v, want *ConflictError", err)
		}
	case errValidation:
		var target *ValidationError
		if !errors.As(err, &target) {
			t.Fatalf("error = %v, want *ValidationError", err)
		}
	default:
		t.Fatalf("unknown error kind %q", want)
	}
}

func TestCheckParticipant(t *testing.T) {
	buyer, seller, stranger := uuid.New(), uuid.New(), uuid.New()
	conv := database.Conversation{BuyerID: buyer, SellerID: seller}

	tests := []struct {
		name   string
		userID uuid.UUID
		want   string
	}{
		{"buyer is a participant", buyer, errNone},
		{"seller is a participant", seller, errNone},
		{"stranger gets 404, not 403", stranger, errNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorKind(t, checkParticipant(conv, tt.userID), tt.want)
		})
	}
}

func TestCheckCanDecide(t *testing.T) {
	buyer, seller, stranger := uuid.New(), uuid.New(), uuid.New()

	conv := func(status string) database.Conversation {
		return database.Conversation{BuyerID: buyer, SellerID: seller, Status: status}
	}

	tests := []struct {
		name   string
		conv   database.Conversation
		userID uuid.UUID
		want   string
	}{
		{"seller decides a pending request", conv(StatusPending), seller, errNone},
		{"buyer may not decide", conv(StatusPending), buyer, errForbidden},
		{"stranger sees a 404, not a 403", conv(StatusPending), stranger, errNotFound},
		{"already accepted", conv(StatusAccepted), seller, errConflict},
		{"already declined", conv(StatusDeclined), seller, errConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorKind(t, checkCanDecide(tt.conv, tt.userID), tt.want)
		})
	}
}

func TestCheckCanSend(t *testing.T) {
	buyer, seller, stranger := uuid.New(), uuid.New(), uuid.New()

	conv := func(status string) database.Conversation {
		return database.Conversation{BuyerID: buyer, SellerID: seller, Status: status}
	}

	tests := []struct {
		name   string
		conv   database.Conversation
		userID uuid.UUID
		want   string
	}{
		{"buyer sends in an accepted thread", conv(StatusAccepted), buyer, errNone},
		{"seller sends in an accepted thread", conv(StatusAccepted), seller, errNone},
		{"nobody sends while pending", conv(StatusPending), buyer, errConflict},
		{"seller cannot send while pending either", conv(StatusPending), seller, errConflict},
		{"declined stays closed", conv(StatusDeclined), buyer, errConflict},
		{"declined stays closed for the seller too", conv(StatusDeclined), seller, errConflict},
		{"stranger is hidden even when accepted", conv(StatusAccepted), stranger, errNotFound},
		{"stranger is hidden while pending", conv(StatusPending), stranger, errNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorKind(t, checkCanSend(tt.conv, tt.userID), tt.want)
		})
	}
}

func TestValidateMessageBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"plain text", "Still available?", errNone},
		{"empty", "", errValidation},
		{"whitespace only", "   \n\t ", errValidation},
		{"at the limit", strings.Repeat("a", maxMessageLength), errNone},
		{"over the limit", strings.Repeat("a", maxMessageLength+1), errValidation},
		{"non-ASCII at the limit", strings.Repeat("ä", maxMessageLength), errNone},
		{"non-ASCII over the limit", strings.Repeat("ä", maxMessageLength+1), errValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateMessageBody(tt.body)
			assertErrorKind(t, err, tt.want)
		})
	}
}

func TestValidateMessageBodyTrims(t *testing.T) {
	got, err := validateMessageBody("  hei  ")
	if err != nil {
		t.Fatalf("unexpected error = %v", err)
	}
	if got != "hei" {
		t.Errorf("body = %q, want %q", got, "hei")
	}
}
