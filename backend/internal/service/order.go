package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/google/uuid"
)

// OrderService holds the business rules for orders.
//
// Note it takes *database.DB, not *database.Queries like ListingService does.
// Queries can only RUN queries; DB can also start transactions (BeginTx), and
// creating an order needs one. DB embeds *Queries, so s.db.GetOrder(...) still
// works for the simple non-transactional reads.
type OrderService struct {
	db *database.DB
}

func NewOrderService(db *database.DB) *OrderService {
	return &OrderService{db: db}
}

// CreateOrder places an order and reserves the stock, atomically.
func (s *OrderService) CreateOrder(ctx context.Context, buyerID uuid.UUID, input dtos.CreateOrderInput) (database.Order, error) {
	// Validate the only two things the client is trusted for.
	if input.ListingID <= 0 {
		return database.Order{}, &ValidationError{Message: "listing_id is required"}
	}
	if input.Quantity <= 0 {
		return database.Order{}, &ValidationError{Message: "quantity must be greater than 0"}
	}

	// nil opts = the database's default isolation level, which is fine here
	// because FOR UPDATE gives us the locking we actually need.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("order transaction rollback failed", "error", err)
	}
	// `defer` runs when the function exits, by ANY path (early return or not).
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Error("order transaction rollback failed", "error", err)
		}
	}()

	// qtx ("queries in transaction") is the same generated API, but every call
	// goes through the transaction instead of the pool. Using s.db here by
	// mistake would silently escape the transaction - that's the one trap.
	qtx := s.db.Queries.WithTx(tx.Tx)

	// FOR UPDATE locks this listing row until we commit. A second buyer running
	// this same query blocks here rather than reading stale stock.
	listing, err := qtx.GetListingForUpdate(ctx, input.ListingID)
	if err != nil {
		// sqlc returns sql.ErrNoRows for a :one query that matched nothing.
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &NotFoundError{Message: "listing not found"}
		}
		return database.Order{}, err
	}

	if listing.SellerID == buyerID {
		return database.Order{}, &ValidationError{Message: "you cannot order your own listing"}
	}
	if listing.Quantity < input.Quantity {
		return database.Order{}, &ConflictError{Message: "not enough stock available"}
	}

	// Decrement first: the query's `AND quantity >= $2` means a race that somehow
	// slipped past the check above matches no rows and errors out, rather than
	// driving stock negative. Defence in depth.
	if _, err := qtx.DecrementListingQuantity(ctx, database.DecrementListingQuantityParams{
		ID:       listing.ID,
		Quantity: input.Quantity,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &ConflictError{Message: "not enough stock available"}
		}
		return database.Order{}, err
	}

	// SellerID and UnitPrice come from the LISTING, never from the request body.
	// listing.Price is a string because the column is NUMERIC(10,2) - pass it
	// straight through; the query computes total_price as unit_price * quantity.
	order, err := qtx.CreateOrder(ctx, database.CreateOrderParams{
		ListingID: listing.ID,
		BuyerID:   buyerID,
		SellerID:  listing.SellerID,
		Quantity:  input.Quantity,
		UnitPrice: listing.Price,
	})
	if err != nil {
		return database.Order{}, err
	}

	// Nothing is real until this line. Before it, another connection sees the
	// old stock and no order at all.
	if err := tx.Commit(); err != nil {
		return database.Order{}, err
	}

	return order, nil
}

// GetOrder returns one order, but only to the two people involved in it.
func (s *OrderService) GetOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	order, err := s.db.GetOrder(ctx, orderID)
	if err != nil {
		return database.Order{}, &NotFoundError{Message: "order not found"}
	}
	if order.BuyerID != userID && order.SellerID != userID {
		return database.Order{}, &ForbiddenError{Message: "you are not part of this order"}
	}
	return order, nil
}

// ListOrders returns every order where the caller is buyer OR seller.
// The query handles both sides, so "my purchases" and "my sales" are one list.
func (s *OrderService) ListOrders(ctx context.Context, userID uuid.UUID) ([]database.Order, error) {
	return s.db.ListOrdersForUser(ctx, userID)
}
