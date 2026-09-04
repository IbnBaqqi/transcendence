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
		{"rating descending", "rating_desc", "COALESCE(sr.average, 0) DESC"},
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

// The join is the expensive half and it exists for one sort. These pin where
// it appears, which no result can show: a query that joins when it need not
// returns exactly the same rows, only slower.
func TestTheRatingJoinAppearsOnlyWhereItIsRead(t *testing.T) {
	const join = "JOIN (SELECT seller_id, AVG(rating)"

	tests := []struct {
		name      string
		sort      string
		countOnly bool
		want      bool
	}{
		{"the rating sort joins", SortRatingDesc, false, true},
		// Sorting cannot change how many rows match, so the count has no
		// reason to pay for the aggregate.
		{"its count query does not", SortRatingDesc, true, false},
		{"another sort does not", "price_asc", false, false},
		{"the default does not", "", false, false},
		// An unknown sort falls back to newest, and the fallback must not drag
		// the join along with it.
		{"an unknown sort does not", "cheapest", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, _ := buildSearchListingsQuery(
				SearchListingsParams{Sort: tt.sort, Limit: 20}, tt.countOnly)

			if got := strings.Contains(query, join); got != tt.want {
				t.Errorf("join present = %v, want %v\ngot: %s", got, tt.want, query)
			}
		})
	}
}

// The aggregate is joined once, not evaluated per row. The per-row form takes
// 55.9s against 14ms on 20k listings because the planner reads every review
// once per listing - and both forms return identical rows, so only the shape
// of the SQL can say which one this is.
func TestTheRatingIsAggregatedNotCorrelated(t *testing.T) {
	query, _ := buildSearchListingsQuery(
		SearchListingsParams{Sort: SortRatingDesc, Limit: 20}, false)

	if !strings.Contains(query, "GROUP BY seller_id) sr ON sr.seller_id = listings.seller_id") {
		t.Errorf("the rating is not a joined aggregate\ngot: %s", query)
	}
	if strings.Contains(query, "ORDER BY (SELECT") {
		t.Errorf("the rating is correlated in ORDER BY, which is quadratic\ngot: %s", query)
	}
}
