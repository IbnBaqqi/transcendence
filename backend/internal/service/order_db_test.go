package service

import (
	"context"
	"errors"
	"testing"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// A 404 test cannot tell "no such order" from "the database is down" - a
// lookup that collapses its error answers both the same way. This one uses a
// real order the caller is party to, so only the distinction can pass it.
func TestADatabaseFailureIsNotReportedAsAMissingOrderToAParty(t *testing.T) {
	f := newOrderFixture(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.svc.GetOrder(ctx, f.buyer, f.order.ID)

	var notFound *NotFoundError
	if errors.As(err, &notFound) {
		t.Fatalf("err = %v, want the underlying failure rather than a 404", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the context cancellation to survive", err)
	}
}

func TestAnOrderThatDoesNotExistIsStillNotFound(t *testing.T) {
	f := newOrderFixture(t)

	_, err := f.svc.GetOrder(context.Background(), f.buyer, database.NewID())

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want NotFoundError", err)
	}
}
