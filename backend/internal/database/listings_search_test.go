package database

import (
	"strings"
	"testing"
)

func TestSortKeysProduceExpectedOrdering(t *testing.T) {
	tests := []struct {
		name      string
		sort      string
		wantOrder string
	}{
		{"empty uses the default", "", "listings.created_at DESC"},
		{"newest", "newest", "listings.created_at DESC"},
		{"oldest", "oldest", "listings.created_at ASC"},
		{"price ascending", "price_asc", "listings.price ASC"},
		{"price descending", "price_desc", "listings.price DESC"},
		{"unknown key falls back", "cheapest", "listings.created_at DESC"},
		{"keys are case sensitive", "PRICE_ASC", "listings.created_at DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := buildSearchListingsQuery(SearchListingsParams{Sort: tt.sort, Limit: 20}, false)

			want := " ORDER BY " + tt.wantOrder + ", listings.id DESC"
			if !strings.Contains(query, want) {
				t.Errorf("missing %q\ngot: %s", want, query)
			}
		})
	}
}

func TestSortKeyNeverReachesSQL(t *testing.T) {
	hostile := "price; DROP TABLE listings--"

	query, args := buildSearchListingsQuery(SearchListingsParams{Sort: hostile, Limit: 20}, false)

	if strings.Contains(query, hostile) || strings.Contains(query, "DROP") {
		t.Fatalf("client input reached the SQL: %s", query)
	}

	def, _ := buildSearchListingsQuery(SearchListingsParams{Sort: "", Limit: 20}, false)
	if query != def {
		t.Errorf("hostile sort changed the query\n got: %s\nwant: %s", query, def)
	}

	if len(args) != 2 {
		t.Errorf("args = %v, want only limit and offset", args)
	}
}

func TestEveryOrderingIsTotal(t *testing.T) {
	for _, key := range SortOptions() {
		query, _ := buildSearchListingsQuery(SearchListingsParams{Sort: key, Limit: 20}, false)
		if !strings.Contains(query, ", listings.id DESC") {
			t.Errorf("sort %q has no tiebreaker: %s", key, query)
		}
	}
}

func TestCountQueryDoesNotSort(t *testing.T) {
	query, _ := buildSearchListingsQuery(SearchListingsParams{Sort: "price_asc"}, true)

	if strings.Contains(query, "ORDER BY") {
		t.Errorf("count query should not sort: %s", query)
	}
}

func TestLikeWildcardsAreEscaped(t *testing.T) {
	query, args := buildSearchListingsQuery(SearchListingsParams{Keyword: "50% off_now", Limit: 20}, false)

	if !strings.Contains(query, "ESCAPE '\\'") {
		t.Errorf("keyword clause has no ESCAPE: %s", query)
	}
	if args[0] != `%50\% off\_now%` {
		t.Errorf("bound keyword = %q, want %q", args[0], `%50\% off\_now%`)
	}

	query, args = buildSearchListingsQuery(SearchListingsParams{Location: `a\b%`, Limit: 20}, false)
	if !strings.Contains(query, "addresses.location ILIKE $1 ESCAPE '\\'") {
		t.Errorf("location clause has no ESCAPE: %s", query)
	}
	if args[0] != `%a\\b\%%` {
		t.Errorf("bound location = %q, want %q", args[0], `%a\\b\%%`)
	}
}

func TestEscapeLike(t *testing.T) {
	tests := []struct{ in, want string }{
		{"chanterelle", "chanterelle"},
		{"50%", `50\%`},
		{"wild_garlic", `wild\_garlic`},
		{`back\slash`, `back\\slash`},
	}

	for _, tt := range tests {
		if got := escapeLike(tt.in); got != tt.want {
			t.Errorf("escapeLike(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsValidSort(t *testing.T) {
	for _, key := range SortOptions() {
		if !IsValidSort(key) {
			t.Errorf("IsValidSort(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"", "cheapest", "PRICE_ASC", "listings.price ASC"} {
		if IsValidSort(key) {
			t.Errorf("IsValidSort(%q) = true, want false", key)
		}
	}
	if !IsValidSort(DefaultSort) {
		t.Error("DefaultSort is not itself a valid key")
	}
}
