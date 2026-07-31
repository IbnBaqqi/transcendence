package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"

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
		return database.Order{}, err
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

// --- Status state machine ---
//
// The whole lifecycle is described by the data below rather than by a pile of
// if-statements. Adding a `refunded` state later means adding one orderAction
// value plus one CHECK constraint - no new logic.

// orderActor says which side of the order may perform a move.
type orderActor int

// iota, just auto-numbers these 0, 1, 2 - tha values don't matter, only that
// they're distinct. It's Go's version of an enum.
const (
	actorSeller orderActor = iota
	actorBuyer
	actorEither
)

// orderAction is one edge of the state machine
type orderAction struct {
	name          string   // used in error messages: "cannot pay an order that is pending"
	from          []string // statuses this move is legal from
	to            string   // status it lands on
	actor         orderActor
	restoresStock bool // hand the reserved quantity back to the listing
}

// The transition table - this IS the lifecycle.
var (
	actionConfirm = orderAction{
		name:  "confirm",
		from:  []string{"pending"},
		to:    "confirmed",
		actor: actorSeller,
	}
	actionPay = orderAction{
		name:  "pay",
		from:  []string{"confirmed"},
		to:    "paid",
		actor: actorBuyer,
	}
	actionComplete = orderAction{
		name:  "complete",
		from:  []string{"paid"},
		to:    "completed",
		actor: actorSeller,
	}
	actionCancel = orderAction{
		name:  "cancel",
		from:  []string{"pending", "confirmed"}, // deliberately NOT "paid" - that needs a refund flow
		to:    "cancelled",
		actor: actorEither,
		// CreateOrder took this quantity out of the listing, so cancelling must
		// put it back or the stock is destroyed forever.
		restoresStock: true,
	}
)

// The four public methods are debliberately thin - every one of them is the same
// operation with a different row from the table above.
func (s *OrderService) ConfirmOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionConfirm)
}

func (s *OrderService) PayOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionPay)
}

func (s *OrderService) CompleteOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionComplete)
}

func (s *OrderService) CancelOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionCancel)
}

// applyAction is the referee: it loads the order, checks who's asking and what
// state it's in, then moves it. Every move runs in a transaction so that a
// cancel's stock restore and its status change can't come apart.
func (s *OrderService) applyAction(ctx context.Context, userID uuid.UUID, orderID int32, action orderAction) (database.Order, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Order{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Error("order status transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	// FOR UPDATE locks the ORDER row (not the listing) until commit. Two
	// simultaneous cancels would otherwise both read "pending", both pass the
	// check below, and both restore the stock - inventing quantity from nowhere.
	order, err := qtx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &NotFoundError{Message: "order not found"}
		}
		return database.Order{}, err
	}

	// Two separate failures, two separate codes: wrong PERSON is 403,
	// wrong STATE is 409. Check the person first - someone with no business
	// here shouldn't learn what state the order is in.
	if err := checkOrderActor(order, userID, action); err != nil {
		return database.Order{}, err
	}

	// slices.Contains reports whether the current status is in the allowed list.
	if !slices.Contains(action.from, order.Status) {
		return database.Order{}, &ConflictError{
			Message: fmt.Sprintf("cannot %s an order that is %s", action.name, order.Status),
		}
	}

	if action.restoresStock {
		if _, err := qtx.IncrementListingQuantity(ctx, database.IncrementListingQuantityParams{
			ID:       order.ListingID,
			Quantity: order.Quantity,
		}); err != nil {
			return database.Order{}, err
		}
	}

	updated, err := qtx.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
		ID:     order.ID,
		Status: action.to,
	})
	if err != nil {
		return database.Order{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.Order{}, err
	}

	return updated, nil
}

// checkOrderActor enforces the "who may travel this edge" guard.
func checkOrderActor(order database.Order, userID uuid.UUID, action orderAction) error {
	isBuyer := order.BuyerID == userID
	isSeller := order.SellerID == userID

	// Matches GetOrder's behaviour: strangers get 403, not 404.
	if !isBuyer && !isSeller {
		return &ForbiddenError{Message: "you are not part of this order"}
	}

	switch action.actor {
	case actorSeller:
		if !isSeller {
			return &ForbiddenError{Message: "only the seller can " + action.name + " this order"}
		}
	case actorBuyer:
		if !isBuyer {
			return &ForbiddenError{Message: "only the buyer can " + action.name + " this order"}
		}
	case actorEither:
		// already covered by the isBuyer/isSeller check above
	}

	return nil
}
