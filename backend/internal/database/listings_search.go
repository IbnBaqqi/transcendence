package database

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

// SearchListingsParams holds already-validated, plain-string filter values.
// Empty string means "no filter" for that field.
type SearchListingsParams struct {
	Keyword  string
	Category string
	Tag      string
	MinPrice string
	MaxPrice string
	Location string
	SellerID string
	// Empty for a signed-out visitor, who cannot have blocked anybody. When
	// set, listings are hidden in BOTH directions: a block conceals each party
	// from the other until it is lifted.
	ViewerID string
	Sort     string
	Offset   int32
	Limit    int32
	// Only ever true for a seller reading their own listings; the service
	// decides that. A sold-out listing is the one that needs restocking or
	// delisting, so hiding it from its owner hides the row they most need.
	IncludeSoldOut bool
}

var sortOptions = map[string]string{
	"newest":     "listings.created_at DESC",
	"oldest":     "listings.created_at ASC",
	"price_asc":  "listings.price ASC",
	"price_desc": "listings.price DESC",
	// Not listings.rating: a rating is per seller, aggregated from reviews, and
	// there is no such column. sr is the join added below for this sort alone.
	//
	// COALESCE puts unrated sellers last under DESC without a special case, so
	// the "new seller" threshold stays a display concern and cannot drift
	// between here and the client.
	SortRatingDesc: "COALESCE(sr.average, 0) DESC",
}

// The one sort that needs a join. Named because two places have to agree about
// it - the map above and the FROM clause below - and a bare string in both is
// how they stop agreeing.
const SortRatingDesc = "rating_desc"

const DefaultSort = "newest"

func IsValidSort(key string) bool {
	_, ok := sortOptions[key]
	return ok
}

func SortOptions() []string {
	keys := make([]string, 0, len(sortOptions))
	for key := range sortOptions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}

func buildSearchListingsQuery(arg SearchListingsParams, countOnly bool) (string, []interface{}) {
	var b strings.Builder
	var args []interface{}

	// Resolved here rather than at the ORDER BY, because the FROM clause has to
	// know: an unknown sort falls back to newest, and a fallback must not drag
	// the join along with it.
	order, ok := sortOptions[arg.Sort]
	if !ok {
		order = sortOptions[DefaultSort]
		arg.Sort = DefaultSort
	}

	// Only for the sort that reads it, and never for the count: sorting cannot
	// change how many rows match, so the count query has no reason to pay for
	// the aggregate.
	//
	// Aggregated once and joined, not computed per listing. A correlated
	// subquery in ORDER BY reads the whole reviews table once per row - 55.9s
	// against 14ms on 20k listings, with idx_reviews_seller_id never used.
	ratingJoin := ""
	if !countOnly && arg.Sort == SortRatingDesc {
		ratingJoin = " LEFT JOIN (SELECT seller_id, AVG(rating) AS average FROM reviews" +
			" GROUP BY seller_id) sr ON sr.seller_id = listings.seller_id"
	}

	next := func(val interface{}) string {
		args = append(args, val)
		return "$" + strconv.Itoa(len(args))
	}

	if countOnly {
		b.WriteString("SELECT COUNT(*) FROM listings LEFT JOIN addresses ON addresses.user_id = listings.seller_id" +
			ratingJoin + " WHERE 1=1")
	} else {
		b.WriteString(`SELECT listings.id, listings.seller_id, listings.title, listings.description,
		listings.category, listings.price, listings.quantity, listings.unit,
		listings.created_at, listings.updated_at
		FROM listings LEFT JOIN addresses ON addresses.user_id = listings.seller_id` + ratingJoin + ` WHERE 1=1`)
	}

	b.WriteString(" AND listings.removed_at IS NULL" +
		" AND EXISTS (SELECT 1 FROM users u WHERE u.id = listings.seller_id AND u.is_visible)")

	// Beside the visibility rules above rather than in a filter below, because
	// it is one: a blocked seller's listings are not a narrower search, they
	// are not there. Above the countOnly split for the same reason SellerID is
	// - otherwise the rows would exclude them while the total still counted
	// them, and every page would come up short.
	if arg.ViewerID != "" {
		p := next(arg.ViewerID)
		b.WriteString(" AND NOT EXISTS (SELECT 1 FROM blocks b" +
			" WHERE (b.blocker_id = " + p + "::uuid AND b.blocked_id = listings.seller_id)" +
			"    OR (b.blocker_id = listings.seller_id AND b.blocked_id = " + p + "::uuid))")
	}

	if !arg.IncludeSoldOut {
		b.WriteString(" AND listings.quantity > 0")
	}

	if arg.Keyword != "" {
		p := next("%" + escapeLike(arg.Keyword) + "%")
		b.WriteString(" AND (listings.title ILIKE " + p + " ESCAPE '\\' OR listings.description ILIKE " + p + " ESCAPE '\\')")
	}
	if arg.Category != "" {
		p := next(arg.Category)
		b.WriteString(" AND listings.category IN (SELECT slug FROM categories" +
			" WHERE slug = " + p + " OR parent_slug = " + p + ")")
	}
	if arg.Tag != "" {
		p := next(arg.Tag)
		b.WriteString(" AND EXISTS (SELECT 1 FROM listing_tags lt" +
			" JOIN tags t ON t.id = lt.tag_id" +
			" WHERE lt.listing_id = listings.id AND t.name = " + p + ")")
	}
	if arg.MinPrice != "" {
		p := next(arg.MinPrice)
		b.WriteString(" AND listings.price::numeric >= " + p + "::numeric")
	}

	if arg.MaxPrice != "" {
		p := next(arg.MaxPrice)
		b.WriteString(" AND listings.price::numeric <= " + p + "::numeric")
	}
	if arg.Location != "" {
		p := next("%" + escapeLike(arg.Location) + "%")
		b.WriteString(" AND addresses.location ILIKE " + p + " ESCAPE '\\'")
	}
	// Above the countOnly split with the other filters: below it, the rows
	// would be one seller's while total counted everybody's.
	if arg.SellerID != "" {
		p := next(arg.SellerID)
		b.WriteString(" AND listings.seller_id = " + p + "::uuid")
	}

	if !countOnly {
		b.WriteString(" ORDER BY " + order + ", listings.id DESC")

		p := next(arg.Limit)
		b.WriteString(" LIMIT " + p)
		p = next(arg.Offset)
		b.WriteString(" OFFSET " + p)
	}

	return b.String(), args
}

func (q *Queries) SearchListingsDynamic(ctx context.Context, arg SearchListingsParams) ([]Listing, error) {
	query, args := buildSearchListingsQuery(arg, false)
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Listing
	for rows.Next() {
		var i Listing
		if err := rows.Scan(
			&i.ID, &i.SellerID, &i.Title, &i.Description, &i.Category,
			&i.Price, &i.Quantity, &i.Unit, &i.CreatedAt, &i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (q *Queries) CountSearchListingsDynamic(ctx context.Context, arg SearchListingsParams) (int64, error) {
	query, args := buildSearchListingsQuery(arg, true)
	var count int64
	err := q.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}
