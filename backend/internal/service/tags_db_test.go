package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/storage"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func tagFixture(t *testing.T) (*ListingService, *database.DB, uuid.UUID) {
	t.Helper()

	db := testdb.New(t)

	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("creating a temporary upload dir: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	seller, err := db.CreateUser(context.Background(), database.CreateUserParams{
		ID:       database.NewID(),
		Username: "seller",
		Email:    "seller@example.test",
		Password: sql.NullString{String: "irrelevant", Valid: true},
	})
	if err != nil {
		t.Fatalf("creating the seller: %v", err)
	}

	return NewListingService(db, files), db, seller.ID
}

func createTagged(
	t *testing.T,
	s *ListingService,
	seller uuid.UUID,
	title string,
	tags []string,
) database.Listing {
	t.Helper()

	listing, err := s.CreateListing(context.Background(), seller, dtos.CreateListingInput{
		Title: title, Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
		Tags: tags,
	})
	if err != nil {
		t.Fatalf("creating %s: %v", title, err)
	}
	return listing
}

func TestTagsSurviveARoundTrip(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles",
		[]string{"SUNNY", "  roadside  ", "Chanterelle", "chanterelle", ""})

	got, err := db.ListTagsForListing(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the tags back: %v", err)
	}

	want := []string{"chanterelle", "roadside", "sunny"}
	if len(got) != len(want) {
		t.Fatalf("tags = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tags = %q, want %q", got, want)
		}
	}
}

func TestUpdatingWithoutTagsClearsThem(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})

	if _, err := listings.UpdateListing(ctx, seller, created.ID, dtos.UpdateListingInput{
		Title: "Chanterelles", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("updating: %v", err)
	}

	got, err := db.ListTagsForListing(ctx, created.ID)
	if err != nil {
		t.Fatalf("reading the tags back: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("tags = %q, want none - PUT replaces the whole set", got)
	}
}

func TestATagOutlivesTheListingThatUsedIt(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	first := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})
	second := createTagged(t, listings, seller, "Morels", []string{"roadside"})

	if err := listings.DeleteListing(ctx, seller, first.ID); err != nil {
		t.Fatalf("deleting the first listing: %v", err)
	}

	got, err := db.ListTagsForListing(ctx, second.ID)
	if err != nil {
		t.Fatalf("reading the survivor's tags: %v", err)
	}
	if len(got) != 1 || got[0] != "roadside" {
		t.Errorf("the survivor's tags = %q, want [roadside] - deleting one listing took the tag with it", got)
	}
}

func TestSearchNormalisesTheTagFilter(t *testing.T) {
	listings, _, seller := tagFixture(t)
	ctx := context.Background()

	createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})
	createTagged(t, listings, seller, "Bilberries", nil)

	tests := []struct {
		name  string
		tag   string
		want  int
		total int64
	}{
		{"exactly as stored", "roadside", 1, 1},
		{"upper case and padded", "  Roadside  ", 1, 1},
		{"a tag nobody uses", "nonsense", 0, 0},
		{"whitespace only must not drop the filter", "   ", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := listings.SearchListings(ctx, dtos.ListingSearchQuery{Tag: tt.tag})
			if err != nil {
				t.Fatalf("searching: %v", err)
			}
			if len(page.Items) != tt.want {
				t.Errorf("items = %d, want %d", len(page.Items), tt.want)
			}
			if page.Total != tt.total {
				t.Errorf("total = %d, want %d", page.Total, tt.total)
			}
		})
	}
}

func TestSearchResultsCarryTheirTags(t *testing.T) {
	listings, _, seller := tagFixture(t)
	ctx := context.Background()

	createTagged(t, listings, seller, "Chanterelles", []string{"roadside", "sunny"})

	page, err := listings.SearchListings(ctx, dtos.ListingSearchQuery{Tag: "roadside"})
	if err != nil {
		t.Fatalf("searching: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}

	got := page.Items[0].Tags
	if len(got) != 2 || got[0] != "roadside" || got[1] != "sunny" {
		t.Errorf("tags on the search result = %q, want [roadside sunny]", got)
	}
}

func TestUpsertTagSurvivesAConcurrentInsertOfTheSameName(t *testing.T) {
	_, db, _ := tagFixture(t)
	ctx := context.Background()

	// A competing transaction claims the name and holds it uncommitted.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("opening the competing transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := db.Queries.WithTx(tx.Tx).UpsertTag(ctx, "chanterelle"); err != nil {
		t.Fatalf("the first upsert: %v", err)
	}

	type result struct {
		id  int32
		err error
	}
	done := make(chan result, 1)
	go func() {
		id, err := db.UpsertTag(ctx, "chanterelle")
		done <- result{id: id, err: err}
	}()

	// Give the racing upsert time to reach the conflicting row before the
	// winner commits; without the row lock it returns immediately instead.
	time.Sleep(300 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatalf("committing: %v", err)
	}

	got := <-done
	if got.err != nil {
		t.Fatalf("a concurrent upsert of the same tag failed: %v - a bare ErrNoRows here reaches the client as a 500", got.err)
	}
	if got.id == 0 {
		t.Error("the racing upsert returned no id")
	}
}

func countTags(t *testing.T, db *database.DB, name string) int {
	t.Helper()

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM tags WHERE name = $1`, name).Scan(&n); err != nil {
		t.Fatalf("counting %s: %v", name, err)
	}
	return n
}

func TestASweepRemovesATagNothingUses(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})

	if _, err := listings.UpdateListing(ctx, seller, created.ID, dtos.UpdateListingInput{
		Title: "Chanterelles", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("dropping the tag: %v", err)
	}

	deleted, err := NewTagService(db).SweepUnused(ctx)
	if err != nil {
		t.Fatalf("sweeping: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1", deleted)
	}
	if got := countTags(t, db, "roadside"); got != 0 {
		t.Errorf("roadside rows = %d, want 0", got)
	}
}

func TestASweepKeepsATagAnotherListingStillUses(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	first := createTagged(t, listings, seller, "Chanterelles", []string{"roadside", "sunny"})
	second := createTagged(t, listings, seller, "Morels", []string{"roadside"})

	if _, err := listings.UpdateListing(ctx, seller, first.ID, dtos.UpdateListingInput{
		Title: "Chanterelles", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("dropping the first listing's tags: %v", err)
	}

	deleted, err := NewTagService(db).SweepUnused(ctx)
	if err != nil {
		t.Fatalf("sweeping: %v", err)
	}
	if deleted != 1 {
		t.Errorf("deleted = %d, want 1 - only sunny is unused", deleted)
	}

	got, err := db.ListTagsForListing(ctx, second.ID)
	if err != nil {
		t.Fatalf("reading the survivor's tags: %v", err)
	}
	if len(got) != 1 || got[0] != "roadside" {
		t.Errorf("the survivor's tags = %q, want [roadside] - the sweep took a tag still in use", got)
	}
}

func TestASweepCollectsATagOrphanedByADeletedListing(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})

	if err := listings.DeleteListing(ctx, seller, created.ID); err != nil {
		t.Fatalf("deleting the listing: %v", err)
	}

	if _, err := NewTagService(db).SweepUnused(ctx); err != nil {
		t.Fatalf("sweeping: %v", err)
	}
	if got := countTags(t, db, "roadside"); got != 0 {
		t.Errorf("roadside rows = %d, want 0", got)
	}
}

func TestAConcurrentAdoptSurvivesTheSweep(t *testing.T) {
	listings, db, seller := tagFixture(t)
	ctx := context.Background()

	created := createTagged(t, listings, seller, "Chanterelles", []string{"roadside"})
	if _, err := listings.UpdateListing(ctx, seller, created.ID, dtos.UpdateListingInput{
		Title: "Chanterelles", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
	}); err != nil {
		t.Fatalf("dropping the tag: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("opening the sweep transaction: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := db.Queries.WithTx(tx.Tx)
	if err := qtx.LockTagsForSweep(ctx); err != nil {
		t.Fatalf("taking the sweep lock: %v", err)
	}

	type adopt struct {
		listing database.Listing
		err     error
	}
	done := make(chan adopt, 1)
	go func() {
		listing, err := listings.CreateListing(ctx, seller, dtos.CreateListingInput{
			Title: "Morels", Category: "mushrooms", Price: 18.00, Quantity: 4, Unit: "kg",
			Tags: []string{"roadside"},
		})
		done <- adopt{listing: listing, err: err}
	}()

	time.Sleep(300 * time.Millisecond)

	select {
	case got := <-done:
		t.Fatalf("the adopt finished while the sweep held the lock: %v", got.err)
	default:
	}

	if _, err := qtx.DeleteUnusedTags(ctx); err != nil {
		t.Fatalf("deleting: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("committing the sweep: %v", err)
	}

	got := <-done
	if got.err != nil {
		t.Fatalf("the blocked adopt failed: %v", got.err)
	}

	if n := countTags(t, db, "roadside"); n != 1 {
		t.Errorf("roadside rows = %d, want 1 - the sweep and the adopt did not agree", n)
	}

	tags, err := db.ListTagsForListing(ctx, got.listing.ID)
	if err != nil {
		t.Fatalf("reading the adopter's tags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "roadside" {
		t.Errorf("the adopter's tags = %q, want [roadside] - the cascade took the attachment", tags)
	}
}
