// Package main seeds the database with realistic sample data for local
// development and demos.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/IbnBaqqi/transcendence/internal/config"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/logger"
	"github.com/google/uuid"
)

// seedUser is just a plain data holder (no methods) describing one seller
// account we want to insert. Struct literals like this are a common Go way
// to define a small "shape" of data.
type seedUser struct {
	Email    string
	Username string
}

// seedListing describes one listing to insert. SellerIdx points at the
// index of the seller in the seedUsers slice below, so we don't need to
// know the real DB id until after the user is inserted.
type seedListing struct {
	SellerIdx   int
	Title       string
	Description string
	Category    string
	Price       string // NUMERIC in Postgres maps to a string in sqlc, not float
	Quantity    int32
	Unit        string
}

// var ... = []T{...} is a package-level slice literal: the sample data
// itself lives here so it's easy to find and tweak later.
var seedUsers = []seedUser{
	{Email: "willow.creek@example.com", Username: "willow-creek"},
	{Email: "moss.harlan@example.com", Username: "moss-harlan"},
	{Email: "river.thorne@example.com", Username: "river-thorne"},
}

var seedListings = []seedListing{
	{SellerIdx: 0, Title: "Golden Chanterelles", Description: "Freshly foraged this morning in the coastal pine forest.", Category: "mushrooms", Price: "18.00", Quantity: 4, Unit: "lb"},
	{SellerIdx: 0, Title: "Wild Morels", Description: "Hand-picked from a recent burn site, excellent flavor.", Category: "mushrooms", Price: "32.00", Quantity: 2, Unit: "lb"},
	{SellerIdx: 0, Title: "Foraged Fiddleheads", Description: "Tightly curled ostrich fern fiddleheads, spring harvest.", Category: "greens", Price: "9.50", Quantity: 6, Unit: "lb"},
	{SellerIdx: 1, Title: "Wild Blueberries", Description: "Small, sweet lowbush blueberries from a sunny hillside.", Category: "berries", Price: "7.00", Quantity: 10, Unit: "lb"},
	{SellerIdx: 1, Title: "Blackberries", Description: "Ripe wild blackberries picked along the river trail.", Category: "berries", Price: "6.50", Quantity: 8, Unit: "lb"},
	{SellerIdx: 1, Title: "Stinging Nettles", Description: "Young nettle tops, great for soups and teas.", Category: "greens", Price: "5.00", Quantity: 5, Unit: "lb"},
	{SellerIdx: 2, Title: "Lion's Mane Mushroom", Description: "Found growing on a downed hardwood, very fresh.", Category: "mushrooms", Price: "22.00", Quantity: 3, Unit: "lb"},
	{SellerIdx: 2, Title: "Wild Ramps", Description: "Foraged in small batches to keep the patch healthy.", Category: "vegetables", Price: "15.00", Quantity: 3, Unit: "bunch"},
	{SellerIdx: 2, Title: "Rose Hips", Description: "Sun-ripened rose hips, hand-cleaned and ready for tea.", Category: "other", Price: "4.50", Quantity: 4, Unit: "lb"},
}

func main() {
	// Keep main() tiny: just call run() and turn any error into a clean
	// exit code. Same pattern as cmd/server/main.go.
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Same three startup steps as the server: load env vars, set up
	// logging, connect to Postgres.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	log, err := logger.New(cfg.Logger.Level, cfg.Env)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// context.Background() is the "empty root" context — we don't need
	// cancellation/timeouts here like the server does for requests.
	ctx := context.Background()

	db, err := database.Connect(ctx, &cfg.DB)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	// defer schedules db.Close() to run when run() returns, no matter
	// which return statement triggers it.
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database connection", "error", err)
		}
	}()

	// Only wipe existing rows in dev — this is what makes `make seed`
	// safe to run repeatedly without piling up duplicates. RESTART
	// IDENTITY resets the auto-increment id counters back to 1; CASCADE
	// is needed because listings has a foreign key to users.
	if cfg.Env == "dev" {
		log.Info("clearing existing listings and users (dev only)")
		if _, err := db.ExecContext(ctx, "TRUNCATE listings, users RESTART IDENTITY CASCADE"); err != nil {
			return fmt.Errorf("failed to truncate tables: %w", err)
		}
	}

	// Insert each seed user, and remember the real DB id Postgres
	// assigned it (sellerIDs[0] is willow's id, etc.) so listings below
	// can reference the right seller.
	sellerIDs := make([]uuid.UUID, 0, len(seedUsers))
	for _, u := range seedUsers {
		user, err := db.Queries.CreateUser(ctx, database.CreateUserParams{
			Email:    u.Email,
			Username: u.Username,
			Password: "seed-placeholder-password", // auth/hashing isn't built yet (#32/#33)
		})
		if err != nil {
			return fmt.Errorf("failed to create seed user %s: %w", u.Email, err)
		}
		sellerIDs = append(sellerIDs, user.ID)
	}

	listingCount := 0
	for _, l := range seedListings {
		_, err := db.Queries.CreateListing(ctx, database.CreateListingParams{
			SellerID: sellerIDs[l.SellerIdx],
			Title:    l.Title,
			// Description column is nullable, so sqlc generates
			// sql.NullString instead of plain string. Valid: true
			// means "this isn't NULL, use the String value".
			Description: sql.NullString{String: l.Description, Valid: true},
			Category:    l.Category,
			Price:       l.Price,
			Quantity:    l.Quantity,
			Unit:        l.Unit,
		})
		if err != nil {
			return fmt.Errorf("failed to create seed listing %q: %w", l.Title, err)
		}
		listingCount++
	}

	log.Info("seed complete", "users", len(sellerIDs), "listings", listingCount)
	return nil
}
