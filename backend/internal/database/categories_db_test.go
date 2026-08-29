package database_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/testdb"
)

func TestTheCategoryTreeCannotGrowAThirdLevel(t *testing.T) {
	db := testdb.New(t)

	if _, err := db.Exec(
		`INSERT INTO categories (slug, name, parent_slug) VALUES ('chanterelles', 'Chanterelles', 'mushrooms')`,
	); err != nil {
		t.Fatalf("a child of a top-level category should be allowed: %v", err)
	}

	refused := []struct {
		name string
		sql  string
	}{
		{
			"a grandchild",
			`INSERT INTO categories (slug, name, parent_slug) VALUES ('golden', 'Golden', 'chanterelles')`,
		},
		{
			"re-parenting a child under another child",
			`INSERT INTO categories (slug, name, parent_slug) VALUES ('morels', 'Morels', 'mushrooms');
			 UPDATE categories SET parent_slug = 'chanterelles' WHERE slug = 'morels'`,
		},
		{
			"demoting a parent that has children",
			`UPDATE categories SET parent_slug = 'berries' WHERE slug = 'mushrooms'`,
		},
		{
			"deleting a parent that has children",
			`DELETE FROM categories WHERE slug = 'mushrooms'`,
		},
		{
			"a category that is its own parent",
			`INSERT INTO categories (slug, name, parent_slug) VALUES ('loop', 'Loop', 'loop')`,
		},
		{
			"a parent that does not exist",
			`INSERT INTO categories (slug, name, parent_slug) VALUES ('orphan', 'Orphan', 'nothing')`,
		},
		{
			"a slug that is not lower case",
			`INSERT INTO categories (slug, name) VALUES ('Truffles', 'Truffles')`,
		},
		{
			"an empty slug",
			`INSERT INTO categories (slug, name) VALUES ('', 'Nameless')`,
		},
	}

	for _, tt := range refused {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := db.Exec(tt.sql); err == nil {
				t.Fatal("the database accepted it")
			}
		})
	}
}

func TestACategoryInUseCannotBeDeleted(t *testing.T) {
	db := testdb.New(t)

	seller := uuid.New()
	if _, err := db.Exec(
		`INSERT INTO users (id, email, username, password) VALUES ($1, 'aino@example.test', 'aino', 'x')`,
		seller,
	); err != nil {
		t.Fatalf("creating the seller: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO listings (id, seller_id, title, description, category, price, quantity, unit)
		 VALUES ($1, $2, 'Bilberries', 'fresh', 'berries', 10.00, 5, 'kg')`,
		database.NewID(), seller,
	); err != nil {
		t.Fatalf("creating the listing: %v", err)
	}

	if _, err := db.Exec(`DELETE FROM categories WHERE slug = 'berries'`); err == nil {
		t.Fatal("a category with listings was deleted, leaving them uncategorised")
	}
}

func TestASlugAlwaysFitsTheColumnItIsRolledBackInto(t *testing.T) {
	db := testdb.New(t)

	long := strings.Repeat("a", 51)
	if _, err := db.Exec(
		`INSERT INTO categories (slug, name) VALUES ($1, 'Too long')`, long,
	); err == nil {
		t.Fatal("a slug over 50 characters was accepted; the Down migration cannot fit it back into varchar(50)")
	}
}
