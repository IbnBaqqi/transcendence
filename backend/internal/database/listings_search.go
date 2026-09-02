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
	Sort     string
	Offset   int32
	Limit    int32
}

var sortOptions = map[string]string{
	"newest":     "listings.created_at DESC",
	"oldest":     "listings.created_at ASC",
	"price_asc":  "listings.price ASC",
	"price_desc": "listings.price DESC",
	// "rating_desc": "listings.rating DESC" — one line, once ratings exist
}

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

	next := func(val interface{}) string {
		args = append(args, val)
		return "$" + strconv.Itoa(len(args))
	}

	if countOnly {
		b.WriteString("SELECT COUNT(*) FROM listings LEFT JOIN addresses ON addresses.user_id = listings.seller_id WHERE 1=1")
	} else {
		b.WriteString(`SELECT listings.id, listings.seller_id, listings.title, listings.description,
		listings.category, listings.price, listings.quantity, listings.unit,
		listings.created_at, listings.updated_at
		FROM listings LEFT JOIN addresses ON addresses.user_id = listings.seller_id WHERE 1=1`)
	}

	b.WriteString(" AND listings.quantity > 0 AND listings.removed_at IS NULL" +
		" AND EXISTS (SELECT 1 FROM users u WHERE u.id = listings.seller_id AND u.is_visible)")

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
		order, ok := sortOptions[arg.Sort]
		if !ok {
			order = sortOptions[DefaultSort]
		}
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
