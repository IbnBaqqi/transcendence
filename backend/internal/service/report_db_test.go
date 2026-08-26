package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

type reportFixture struct {
	svc      *ReportService
	db       *database.DB
	seller   uuid.UUID
	reporter uuid.UUID
	other    uuid.UUID
	listing  uuid.UUID
}

func newReportFixture(t *testing.T) reportFixture {
	t.Helper()

	db := testdb.New(t)
	ctx := context.Background()

	mk := func(name string) uuid.UUID {
		user, err := db.CreateUser(ctx, database.CreateUserParams{
			ID:       database.NewID(),
			Username: name, Email: name + "@example.test", Password: "irrelevant",
		})
		if err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
		return user.ID
	}
	seller, reporter, other := mk("seller"), mk("reporter"), mk("other")

	listing, err := db.CreateListing(ctx, database.CreateListingParams{
		ID:       database.NewID(),
		SellerID: seller, Title: "Chanterelles", Category: "mushrooms",
		Price: "18.10", Quantity: 5, Unit: "kg",
	})
	if err != nil {
		t.Fatalf("creating a listing: %v", err)
	}

	return reportFixture{
		svc: NewReportService(db.Queries), db: db,
		seller: seller, reporter: reporter, other: other, listing: listing.ID,
	}
}

func (f reportFixture) report(t *testing.T, reporter uuid.UUID, reason, detail string) error {
	t.Helper()
	return f.svc.Report(context.Background(), reporter, f.listing, reason, detail)
}

func TestAReportLandsInTheQueueAsOpen(t *testing.T) {
	f := newReportFixture(t)

	if err := f.report(t, f.reporter, "spam", ""); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	var status string
	var detail sql.NullString
	if err := f.db.QueryRow(
		`SELECT status, detail FROM listing_reports`,
	).Scan(&status, &detail); err != nil {
		t.Fatalf("reading the report: %v", err)
	}

	if status != "open" {
		t.Errorf("status = %q, want open", status)
	}
	if detail.Valid {
		t.Errorf("detail = %q, want NULL", detail.String)
	}
}

func TestReportingDoesNotTouchTheListing(t *testing.T) {
	f := newReportFixture(t)
	ctx := context.Background()

	before, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("reading the listing: %v", err)
	}

	if err := f.report(t, f.reporter, "prohibited", "not legal to sell"); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	after, err := f.db.GetListing(ctx, f.listing)
	if err != nil {
		t.Fatalf("the listing disappeared after a report: %v", err)
	}
	if after.Quantity != before.Quantity || after.Title != before.Title {
		t.Error("the listing changed when it was reported")
	}
}

func TestYouCannotReportYourOwnListing(t *testing.T) {
	f := newReportFixture(t)

	err := f.report(t, f.seller, "spam", "")

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %#v, want *ValidationError", err)
	}
}

func TestASecondReportFromTheSamePersonIsRefused(t *testing.T) {
	f := newReportFixture(t)

	if err := f.report(t, f.reporter, "spam", ""); err != nil {
		t.Fatalf("first report: %v", err)
	}

	err := f.report(t, f.reporter, "offensive", "changed my mind")

	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("err = %#v, want *ConflictError", err)
	}
}

func TestADifferentPersonCanReportTheSameListing(t *testing.T) {
	f := newReportFixture(t)

	if err := f.report(t, f.reporter, "spam", ""); err != nil {
		t.Fatalf("first report: %v", err)
	}
	if err := f.report(t, f.other, "misleading", ""); err != nil {
		t.Fatalf("second reporter refused: %v", err)
	}

	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM listing_reports`).Scan(&n); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if n != 2 {
		t.Errorf("reports = %d, want 2", n)
	}
}

func TestADetailWithANullByteIsRefused(t *testing.T) {
	f := newReportFixture(t)

	err := f.report(t, f.reporter, "spam", "a\x00b")

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %#v, want *ValidationError - Postgres cannot store NUL, so anything else is a 500", err)
	}
}

func TestDetailLengthIsCountedInCharacters(t *testing.T) {
	f := newReportFixture(t)

	detail := strings.Repeat("ä", 400)

	if err := f.report(t, f.reporter, "other", detail); err != nil {
		t.Fatalf("a 400-character detail was refused: %v", err)
	}

	var stored string
	if err := f.db.QueryRow(`SELECT detail FROM listing_reports`).Scan(&stored); err != nil {
		t.Fatalf("reading the detail back: %v", err)
	}
	if utf8.RuneCountInString(stored) != 400 {
		t.Errorf("stored %d characters, want 400", utf8.RuneCountInString(stored))
	}
}

func TestDetailIsStrippedOfControlCharacters(t *testing.T) {
	f := newReportFixture(t)

	if err := f.report(t, f.reporter, "other", "red \x1b[31mtext\x1b[0m and ‮reversed"); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	var stored string
	if err := f.db.QueryRow(`SELECT detail FROM listing_reports`).Scan(&stored); err != nil {
		t.Fatalf("reading the detail back: %v", err)
	}

	if strings.ContainsRune(stored, 0x1b) {
		t.Errorf("an ANSI escape reached the moderator: %q", stored)
	}
	if strings.ContainsRune(stored, '‮') {
		t.Errorf("a bidi override reached the moderator: %q", stored)
	}
	if !strings.Contains(stored, "red") || !strings.Contains(stored, "reversed") {
		t.Errorf("the readable text was destroyed: %q", stored)
	}
}

func TestAnUnknownReasonIsRefusedBeforeTheDatabase(t *testing.T) {
	f := newReportFixture(t)

	err := f.report(t, f.reporter, "because-i-said-so", "")

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("err = %#v, want *ValidationError", err)
	}
}

func TestReportingAnUnknownListingIsNotFound(t *testing.T) {
	f := newReportFixture(t)

	err := f.svc.Report(context.Background(), f.reporter, uuid.New(), "spam", "")

	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %#v, want *NotFoundError", err)
	}
}

func TestAReportSurvivesItsReporterLeaving(t *testing.T) {
	f := newReportFixture(t)
	ctx := context.Background()

	if err := f.report(t, f.reporter, "spam", ""); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	if err := f.db.DeleteUser(ctx, f.reporter); err != nil {
		t.Fatalf("deleting the reporter: %v", err)
	}

	var total, withReporter int
	if err := f.db.QueryRow(
		`SELECT count(*), count(reporter_id) FROM listing_reports`,
	).Scan(&total, &withReporter); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if total != 1 {
		t.Errorf("reports = %d, want the report to outlive its reporter", total)
	}
	if withReporter != 0 {
		t.Error("the reporter was not anonymised")
	}
}

func TestReportsGoWithTheListing(t *testing.T) {
	f := newReportFixture(t)

	if err := f.report(t, f.reporter, "spam", ""); err != nil {
		t.Fatalf("reporting: %v", err)
	}

	if _, err := f.db.Exec(`DELETE FROM listings WHERE id = $1`, f.listing); err != nil {
		t.Fatalf("deleting the listing: %v", err)
	}

	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM listing_reports`).Scan(&n); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if n != 0 {
		t.Errorf("reports = %d, want 0", n)
	}
}
