package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

func TestSearchListingsRejectsUnknownSort(t *testing.T) {
	svc := NewListingService(nil)

	_, err := svc.SearchListings(context.Background(), dtos.ListingSearchQuery{Sort: "cheapest"})

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
	if !strings.Contains(validation.Message, "price_asc") {
		t.Errorf("message %q should list the valid options", validation.Message)
	}
}
