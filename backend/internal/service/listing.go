package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/google/uuid"
)

// ListingService contains the business logic for listings: validation,
// ownership rules, and orchestration of database calls.
type ListingService struct {
	db    *database.DB
	files fileStore
}

func NewListingService(db *database.DB, files fileStore) *ListingService {
	return &ListingService{db: db, files: files}
}

const listingCategoryConstraint = "listings_category_fkey"

func normaliseCategory(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// Rune-counted below, since varchar(20) counts characters rather than bytes.
const (
	maxUnitLength        = 20
	maxDescriptionLength = 1024 // mirrors descriptionSchema; the column is bare text
)

func validateListingInput(title, description, category, unit string, price float64, quantity int32) error {
	if title == "" || len(title) > 100 {
		return &ValidationError{Message: "Title is required and must be under 100 characters"}
	}
	// The column is bare text, so without this the only bound on a description
	// is the request body cap.
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		return &ValidationError{
			Message: fmt.Sprintf("Description must be under %d characters", maxDescriptionLength),
		}
	}
	if category == "" {
		return &ValidationError{Message: "Category is required"}
	}
	// Over the varchar(20) column a unit reached Postgres and came back a 500.
	if unit == "" || utf8.RuneCountInString(unit) > maxUnitLength {
		return &ValidationError{
			Message: fmt.Sprintf("Unit is required and must be under %d characters", maxUnitLength),
		}
	}
	if price <= 0 {
		return &ValidationError{Message: "Price must be greater than 0"}
	}
	if quantity <= 0 {
		return &ValidationError{Message: "Quantity must be greater than 0"}
	}
	return nil
}

func normaliseTags(raw []string) ([]string, error) {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))

	for _, tag := range raw {
		if !utf8.ValidString(tag) || strings.ContainsRune(tag, 0) {
			return nil, &ValidationError{Message: "Tags must be valid UTF-8 without null bytes"}
		}

		tag = strings.ToLower(strings.TrimSpace(sanitizeFreeText(tag)))
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > maxTagLength {
			return nil, &ValidationError{Message: "A tag is too long"}
		}
		if seen[tag] {
			continue
		}

		seen[tag] = true
		out = append(out, tag)
	}

	if len(out) > maxTagsPerListing {
		return nil, &ValidationError{Message: "A listing can have at most 5 tags"}
	}

	// Do not remove: UpsertTag's ON CONFLICT DO UPDATE locks each existing tag
	// row until commit, so two listings saved at once with overlapping tags in
	// different orders ({a,b} against {b,a}) deadlock. Sorting gives every
	// transaction the same lock order.
	slices.Sort(out)

	return out, nil
}

func applyTags(ctx context.Context, qtx *database.Queries, listingID uuid.UUID, tags []string) error {
	if err := qtx.LockTagsShared(ctx); err != nil {
		return err
	}

	if err := qtx.DetachAllTags(ctx, listingID); err != nil {
		return err
	}

	for _, name := range tags {
		tagID, err := qtx.UpsertTag(ctx, name)
		if err != nil {
			return err
		}
		if err := qtx.AttachTag(ctx, database.AttachTagParams{
			ListingID: listingID,
			TagID:     tagID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *ListingService) CreateListing(ctx context.Context, sellerID uuid.UUID, input dtos.CreateListingInput) (database.Listing, error) {
	category := normaliseCategory(input.Category)

	if err := validateListingInput(input.Title, input.Description, category, input.Unit, input.Price, input.Quantity); err != nil {
		return database.Listing{}, err
	}

	tags, err := normaliseTags(input.Tags)
	if err != nil {
		return database.Listing{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Listing{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("listing create transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	listing, err := qtx.CreateListing(ctx, database.CreateListingParams{
		ID:          database.NewID(),
		SellerID:    sellerID,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
	if isForeignKeyViolation(err, listingCategoryConstraint) {
		return database.Listing{}, &ValidationError{Message: "Category is not recognised"}
	}
	if err != nil {
		return database.Listing{}, err
	}

	if err := applyTags(ctx, qtx, listing.ID, tags); err != nil {
		return database.Listing{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.Listing{}, err
	}

	return listing, nil
}

func (s *ListingService) GetListing(ctx context.Context, id uuid.UUID) (database.Listing, error) {
	listing, err := s.db.GetListing(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Listing{}, &NotFoundError{Message: "Listing not found"}
		}
		return database.Listing{}, err
	}
	return listing, nil
}

// UpdateListing edits a listing the caller owns.
func (s *ListingService) UpdateListing(ctx context.Context, userID uuid.UUID, listingID uuid.UUID, input dtos.UpdateListingInput) (database.Listing, error) {
	category := normaliseCategory(input.Category)

	if err := validateListingInput(input.Title, input.Description, category, input.Unit, input.Price, input.Quantity); err != nil {
		return database.Listing{}, err
	}

	tags, err := normaliseTags(input.Tags)
	if err != nil {
		return database.Listing{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Listing{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("listing update transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	existing, err := qtx.GetListingForUpdate(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Listing{}, &NotFoundError{Message: "Listing not found"}
		}
		return database.Listing{}, err
	}

	if existing.SellerID != userID {
		return database.Listing{}, &ForbiddenError{Message: "You do not own this listing"}
	}

	if existing.Quantity == 0 {
		return database.Listing{}, &ConflictError{Message: "Listing is sold out and can no longer be edited; create new listing"}
	}

	updated, err := qtx.UpdateListing(ctx, database.UpdateListingParams{
		ID:          listingID,
		Title:       input.Title,
		Description: sql.NullString{String: input.Description, Valid: input.Description != ""},
		Category:    category,
		Price:       strconv.FormatFloat(input.Price, 'f', 2, 64),
		Quantity:    input.Quantity,
		Unit:        input.Unit,
	})
	if err != nil {
		if isForeignKeyViolation(err, listingCategoryConstraint) {
			return database.Listing{}, &ValidationError{Message: "Category is not recognised"}
		}
		return database.Listing{}, err
	}

	if err := applyTags(ctx, qtx, listingID, tags); err != nil {
		return database.Listing{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.Listing{}, err
	}

	return updated, nil
}

// DeleteListing removes a listing the caller owns.
func (s *ListingService) DeleteListing(ctx context.Context, userID uuid.UUID, listingID uuid.UUID) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("listing delete transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	existing, err := qtx.GetListingForUpdate(ctx, listingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Listing not found"}
		}
		return err
	}
	if existing.SellerID != userID {
		return &ForbiddenError{Message: "You do not own this listing"}
	}

	// moderation_actions and listing_reports both cascade from listings, so
	// deleting a removed listing would erase the reports and the record of
	// the decision along with it.
	if existing.RemovedAt.Valid {
		return &ForbiddenError{Message: "This listing was removed by a moderator and cannot be deleted"}
	}

	// Only orders still in flight stand in the way. A finished one does not:
	// orders.listing_id is SET NULL since 024, and every field an order shows is
	// snapshotted on its own row, so it survives the listing intact.
	activeCount, err := qtx.CountActiveOrdersForListing(ctx, uuid.NullUUID{UUID: listingID, Valid: true})
	if err != nil {
		return err
	}
	if activeCount > 0 {
		return &ConflictError{Message: "This listing has a sale in progress — finish or cancel it before deleting"}
	}

	filenames, err := qtx.DeleteImagesForListing(ctx, listingID)
	if err != nil {
		return err
	}

	if err := notifySavers(ctx, qtx, existing, userID, notifyKindSavedListingDeleted); err != nil {
		return err
	}

	if err := qtx.DeleteListing(ctx, listingID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	for _, name := range filenames {
		if err := s.files.Delete(name); err != nil {
			slog.Error("failed to delete image file", "filename", name, "error", err)
		}
	}

	return nil
}

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 50

	maxSearchTextLength = 200

	maxTagsPerListing = 5
	maxTagLength      = 30
)

func resolveSort(sortKey string) (string, error) {
	if sortKey == "" {
		return database.DefaultSort, nil
	}
	if !database.IsValidSort(sortKey) {
		return "", &ValidationError{
			Message: "Sort must be one of: " + strings.Join(database.SortOptions(), ", "),
		}
	}
	return sortKey, nil
}

func validateSearchText(values ...string) error {
	for _, value := range values {
		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return &ValidationError{Message: "Search text must be valid UTF-8 without null bytes"}
		}
		if utf8.RuneCountInString(value) > maxSearchTextLength {
			return &ValidationError{Message: "Search text is too long"}
		}
	}
	return nil
}

// uuid.Nil means nobody is signed in, and the empty string is what the builder
// reads as "no viewer" - a signed-out visitor cannot have blocked anyone.
func viewerString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

func (s *ListingService) SearchListings(ctx context.Context, viewerID uuid.UUID, q dtos.ListingSearchQuery) (dtos.PaginatedListings, error) {
	if err := validateSearchText(q.Keyword, q.Category, q.Tag, q.Location); err != nil {
		return dtos.PaginatedListings{}, err
	}

	// Parsed here rather than left to the ::uuid cast in the query: Postgres
	// answers a malformed value with an error the service cannot tell from a
	// real failure, so it would reach the client as 500 instead of 400.
	var sellerID uuid.UUID
	if q.SellerID != "" {
		parsed, err := uuid.Parse(q.SellerID)
		if err != nil {
			return dtos.PaginatedListings{}, &ValidationError{Message: "Seller id must be a UUID"}
		}
		sellerID = parsed
	}

	page := defaultPage
	if q.Page != "" {
		p, err := strconv.Atoi(q.Page)
		if err != nil || p < 1 || p > math.MaxInt32 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "Page must be a positive integer"}
		}
		page = p
	}

	limit := defaultLimit
	if q.Limit != "" {
		l, err := strconv.Atoi(q.Limit)
		if err != nil || l < 1 {
			return dtos.PaginatedListings{}, &ValidationError{Message: "Limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	var minPrice, maxPrice sql.NullString
	var minVal, maxVal float64

	if q.MinPrice != "" {
		v, err := strconv.ParseFloat(q.MinPrice, 64)
		if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return dtos.PaginatedListings{}, &ValidationError{Message: "Min price must be a non-negative number"}
		}
		minVal = v
		minPrice = sql.NullString{String: strconv.FormatFloat(v, 'f', 2, 64), Valid: true}
	}
	if q.MaxPrice != "" {
		v, err := strconv.ParseFloat(q.MaxPrice, 64)
		if err != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return dtos.PaginatedListings{}, &ValidationError{Message: "Max price must be a non-negative number"}
		}
		maxVal = v
		maxPrice = sql.NullString{String: strconv.FormatFloat(v, 'f', 2, 64), Valid: true}
	}
	if minPrice.Valid && maxPrice.Valid && minVal > maxVal {
		return dtos.PaginatedListings{}, &ValidationError{Message: "Min price must not exceed max_price"}
	}

	sortKey, err := resolveSort(q.Sort)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}

	offset := (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return dtos.PaginatedListings{}, &ValidationError{Message: "Page is too large"}
	}

	tag := ""
	if q.Tag != "" {
		tags, err := normaliseTags([]string{q.Tag})
		if err != nil {
			return dtos.PaginatedListings{}, err
		}
		if len(tags) == 0 {
			return dtos.PaginatedListings{
				Items: []dtos.ListingResponse{}, Page: page, Limit: limit,
			}, nil
		}
		tag = tags[0]
	}

	params := database.SearchListingsParams{
		Keyword:  q.Keyword,
		Category: normaliseCategory(q.Category),
		Tag:      tag,
		Location: q.Location,
		SellerID: q.SellerID,
		ViewerID: viewerString(viewerID),
		Sort:     sortKey,
		Offset:   int32(offset),
		Limit:    int32(limit),
		// Ignored rather than refused when the caller is not the seller: a 400
		// would advertise the parameter and turn a harmless request into an
		// error, while ignoring it means no caller can widen public search.
		// The uuid.Nil guard is what stops an anonymous caller with no
		// seller_id matching itself - Nil == Nil is true.
		IncludeSoldOut: q.IncludeSoldOut == "true" && sellerID != uuid.Nil && sellerID == viewerID,
	}
	if minPrice.Valid {
		params.MinPrice = minPrice.String
	}
	if maxPrice.Valid {
		params.MaxPrice = maxPrice.String
	}

	items, err := s.db.SearchListingsDynamic(ctx, params)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}
	total, err := s.db.CountSearchListingsDynamic(ctx, params)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}

	responses, err := s.Responses(ctx, items)
	if err != nil {
		return dtos.PaginatedListings{}, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return dtos.PaginatedListings{
		Items:      responses,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

// Responses turns listing rows into the shape the API answers with: images,
// tags and the seller are separate tables, so every listing endpoint has to
// attach all three. Doing it here rather than in each handler is what keeps a
// fourth one from being added in five places.
//
// This is why listings differ from orders and conversations, whose services
// return database rows and leave the handler to convert: a listing is the only
// resource assembled from four tables, so the assembly is worth owning in one
// place. Adding a Responses to another service is not the pattern to copy
// unless that resource grows the same problem - #202 has the reasoning.
func (s *ListingService) Responses(
	ctx context.Context,
	rows []database.Listing,
) ([]dtos.ListingResponse, error) {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	byListing, err := imagesByListing(ctx, s.db.Queries, ids)
	if err != nil {
		return nil, err
	}

	tagsByListing, err := s.TagsByListing(ctx, ids)
	if err != nil {
		return nil, err
	}

	bySeller, err := s.SellersFor(ctx, rows)
	if err != nil {
		return nil, err
	}

	return dtos.WithSellerEach(
		dtos.WithTagsEach(dtos.ToListingResponsesWithImages(rows, byListing), tagsByListing),
		bySeller,
	), nil
}

func (s *ListingService) Response(
	ctx context.Context,
	row database.Listing,
) (dtos.ListingResponse, error) {
	responses, err := s.Responses(ctx, []database.Listing{row})
	if err != nil {
		return dtos.ListingResponse{}, err
	}
	// One row in, one out: every decorator maps a listing to a listing rather
	// than filtering, so this index is safe for as long as that stays true.
	return responses[0], nil
}

func (s *ListingService) SellersFor(
	ctx context.Context,
	items []database.Listing,
) (map[uuid.UUID]dtos.ListingSeller, error) {
	out := make(map[uuid.UUID]dtos.ListingSeller, len(items))
	if len(items) == 0 {
		return out, nil
	}

	seen := make(map[uuid.UUID]struct{}, len(items))
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.SellerID]; ok {
			continue
		}
		seen[item.SellerID] = struct{}{}
		ids = append(ids, item.SellerID)
	}

	rows, err := s.db.ListSellersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.ID] = dtos.ToListingSeller(row.ID, row.Username, row.AvatarFilename, row.RatingAverage, row.RatingCount)
	}
	return out, nil
}

func (s *ListingService) TagsForListing(ctx context.Context, id uuid.UUID) ([]string, error) {
	return s.db.ListTagsForListing(ctx, id)
}

func (s *ListingService) TagsByListing(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := s.db.ListTagsForListings(ctx, ids)
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		out[row.ListingID] = append(out[row.ListingID], row.Name)
	}

	return out, nil
}

func (s *ListingService) ListCategories(ctx context.Context) ([]database.ListCategoriesRow, error) {
	return s.db.ListCategories(ctx)
}
