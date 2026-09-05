package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

// #272. A block used to stop messages and nothing else, so the person you
// blocked could still buy from you. It now hides the two of you from each
// other - and because the refusal has to be indistinguishable from a listing
// that was taken down, every one of these is a 404 rather than a 403.
func TestABlockStopsANewOrderInBothDirections(t *testing.T) {
	f := newBlockFixture(t)
	orders := NewOrderService(f.db, notify.Disabled{})
	ctx := context.Background()

	buy := func(buyer, listing uuid.UUID) error {
		_, err := orders.CreateOrder(ctx, buyer, dtos.CreateOrderInput{
			ListingID: listing, Quantity: 1,
		})
		return err
	}

	// The control: without a block this is an ordinary sale.
	if err := buy(f.buyer, f.listing); err != nil {
		t.Fatalf("an unblocked buyer could not order: %v", err)
	}

	if err := f.blocks.Block(ctx, f.seller, f.buyer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	t.Run("the blocked buyer is refused", func(t *testing.T) {
		err := buy(f.buyer, f.listing)
		if !isNotFound(err) {
			t.Fatalf("err = %#v, want *NotFoundError - a 403 would announce the block", err)
		}
	})

	// Symmetric: the direction of the block must not decide who can trade.
	t.Run("and so is the blocker, buying the other way", func(t *testing.T) {
		theirs := newListingBy(t, f.db, f.buyer, "Blueberries")
		if err := buy(f.seller, theirs); !isNotFound(err) {
			t.Fatalf("err = %#v, want *NotFoundError", err)
		}
	})

	t.Run("unblocking lets the sale happen again", func(t *testing.T) {
		if err := f.blocks.Unblock(ctx, f.seller, f.buyer); err != nil {
			t.Fatalf("unblocking: %v", err)
		}
		if err := buy(f.buyer, f.listing); err != nil {
			t.Errorf("after unblocking the buyer still cannot order: %v", err)
		}
	})
}

// The listings have to disappear from search too, or the block is cosmetic:
// the row is still there to click, and only the order refuses.
func TestABlockHidesListingsFromSearchBothWays(t *testing.T) {
	f := newBlockFixture(t)
	listings := NewListingService(f.db, nil)
	ctx := context.Background()

	newListingBy(t, f.db, f.buyer, "Blueberries")

	titles := func(t *testing.T, viewer uuid.UUID) map[string]bool {
		t.Helper()
		page, err := listings.SearchListings(ctx, viewer, dtos.ListingSearchQuery{})
		if err != nil {
			t.Fatalf("searching: %v", err)
		}
		out := map[string]bool{}
		for _, l := range page.Items {
			out[l.Title] = true
		}
		return out
	}

	if got := titles(t, f.buyer); !got["Chanterelles"] {
		t.Fatal("the control failed: the seller's listing is missing before any block")
	}

	if err := f.blocks.Block(ctx, f.seller, f.buyer); err != nil {
		t.Fatalf("blocking: %v", err)
	}

	t.Run("the blocked buyer stops seeing the seller's listings", func(t *testing.T) {
		if titles(t, f.buyer)["Chanterelles"] {
			t.Error("the seller blocked the buyer and their listing is still in the buyer's search")
		}
	})

	t.Run("and the blocker stops seeing theirs", func(t *testing.T) {
		if titles(t, f.seller)["Blueberries"] {
			t.Error("the seller blocked the buyer and still sees the buyer's listing")
		}
	})

	// Nothing is deleted - the rule is a filter, so lifting it restores both.
	t.Run("unblocking brings both back", func(t *testing.T) {
		if err := f.blocks.Unblock(ctx, f.seller, f.buyer); err != nil {
			t.Fatalf("unblocking: %v", err)
		}
		if !titles(t, f.buyer)["Chanterelles"] {
			t.Error("after unblocking the buyer still cannot see the seller's listing")
		}
		if !titles(t, f.seller)["Blueberries"] {
			t.Error("after unblocking the seller still cannot see the buyer's listing")
		}
	})

	// A signed-out visitor has blocked nobody, so the filter must not fire for
	// them - an empty viewer is "no viewer", not "block everything".
	t.Run("a signed-out visitor sees both regardless", func(t *testing.T) {
		if err := f.blocks.Block(ctx, f.seller, f.buyer); err != nil {
			t.Fatalf("re-blocking: %v", err)
		}
		anon := titles(t, uuid.Nil)
		if !anon["Chanterelles"] || !anon["Blueberries"] {
			t.Errorf("a visitor should see both listings, saw %v", anon)
		}
	})
}
