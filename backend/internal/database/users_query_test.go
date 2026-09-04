package database

import (
	"strings"
	"testing"
)

// The predicate inside the aggregate changes no row - the join filters to these
// sellers either way - so nothing that reads results can tell whether it is
// there. What it changes is how much of the reviews table gets read: without it
// every listing page aggregates every review ever written. The query is a
// generated constant, so its text is the only seam that can see the difference.
func TestTheRatingAggregateIsScopedToTheRequestedSellers(t *testing.T) {
	if !strings.Contains(listSellersByIDs, "WHERE seller_id = ANY($1::uuid[])") {
		t.Errorf("the rating aggregate is no longer scoped to the sellers on the page,"+
			" so it reads the whole reviews table\ngot: %s", listSellersByIDs)
	}
}
