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
type OrderService struct {
	db *database.DB
}

func NewOrderService(db *database.DB) *OrderService {
	return &OrderService{db: db}
}

func (s *OrderService) CreateOrder(ctx context.Context, buyerID uuid.UUID, input dtos.CreateOrderInput) (database.Order, error) {
	if input.ListingID <= 0 {
		return database.Order{}, &ValidationError{Message: "listing_id is required"}
	}
	if input.Quantity <= 0 {
		return database.Order{}, &ValidationError{Message: "quantity must be greater than 0"}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Order{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil {
			slog.Error("order transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	listing, err := qtx.GetListingForUpdate(ctx, input.ListingID)
	if err != nil {
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

	if _, err := qtx.DecrementListingQuantity(ctx, database.DecrementListingQuantityParams{
		ID:       listing.ID,
		Quantity: input.Quantity,
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &ConflictError{Message: "not enough stock available"}
		}
		return database.Order{}, err
	}

	order, err := qtx.CreateOrder(ctx, database.CreateOrderParams{
		ListingID:    listing.ID,
		BuyerID:      buyerID,
		SellerID:     listing.SellerID,
		Quantity:     input.Quantity,
		UnitPrice:    listing.Price,
		ListingTitle: listing.Title,
	})
	if err != nil {
		return database.Order{}, err
	}

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
func (s *OrderService) ListOrders(ctx context.Context, userID uuid.UUID) ([]database.Order, error) {
	return s.db.ListOrdersForUser(ctx, userID)
}

// orderActor says which side of the order may perform a move.
type orderActor int

const (
	actorSeller orderActor = iota
	actorBuyer
	actorEither
)

// handshakeMark says which side's confirmation a move records.
type handshakeMark int

const (
	markNone handshakeMark = iota
	markSeller
	markBuyer
)

// orderAction is one edge of the state machine
type orderAction struct {
	name             string
	from             []string
	to               string
	actor            orderActor
	restoresStock    bool
	mark             handshakeMark
	blockedAfterMark bool
}

// The transition table - this IS the lifecycle.
var (
	actionConfirm = orderAction{
		name:  "confirm",
		from:  []string{"pending"},
		to:    "confirmed",
		actor: actorSeller,
	}
	actionHandover = orderAction{
		name:  "hand over",
		from:  []string{"confirmed"},
		to:    "completed",
		actor: actorSeller,
		mark:  markSeller,
	}
	actionReceive = orderAction{
		name:  "confirm receipt of",
		from:  []string{"confirmed"},
		to:    "completed",
		actor: actorBuyer,
		mark:  markBuyer,
	}
	actionCancel = orderAction{
		name:             "cancel",
		from:             []string{"pending", "confirmed"},
		to:               "cancelled",
		actor:            actorEither,
		restoresStock:    true,
		blockedAfterMark: true,
	}
)

func (s *OrderService) ConfirmOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionConfirm)
}

// HandoverOrder records the SELLER's half of the handshake.
func (s *OrderService) HandoverOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionHandover)
}

// ReceiveOrder records the BUYER's half of the handshake.
func (s *OrderService) ReceiveOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionReceive)
}

func (s *OrderService) CancelOrder(ctx context.Context, userID uuid.UUID, orderID int32) (database.Order, error) {
	return s.applyAction(ctx, userID, orderID, actionCancel)
}

// applyAction is the referee: it loads the order, checks who's asking and what
// state it's in, then moves it.
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

	order, err := qtx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &NotFoundError{Message: "order not found"}
		}
		return database.Order{}, err
	}

	if err := checkOrderActor(order, userID, action); err != nil {
		return database.Order{}, err
	}
	// Status first, so an order in the wrong state reports that plainly rather
	// than blaming the handshake.
	if !slices.Contains(action.from, order.Status) {
		return database.Order{}, &ConflictError{
			Message: fmt.Sprintf("cannot %s an order that is %s", action.name, order.Status),
		}
	}

	if err := checkHandshakeLock(order, action); err != nil {
		return database.Order{}, err
	}

	if action.restoresStock {
		if _, err := qtx.IncrementListingQuantity(ctx, database.IncrementListingQuantityParams{
			ID:       order.ListingID,
			Quantity: order.Quantity,
		}); err != nil {
			return database.Order{}, err
		}
	}

	if action.mark != markNone {
		marked, err := markHandshake(ctx, qtx, order, action)
		if err != nil {
			return database.Order{}, err
		}
		if !bothSidesMarked(marked) {
			if err := tx.Commit(); err != nil {
				return database.Order{}, err
			}
			return marked, nil
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

// markHandshake stamps one side's confirmation, refusing a second stamp from
// the same side so a double-click can't look like progress.
func markHandshake(ctx context.Context, qtx *database.Queries, order database.Order, action orderAction) (database.Order, error) {
	switch action.mark {
	case markSeller:
		if order.SellerHandedOverAt.Valid {
			return database.Order{}, &ConflictError{Message: "you have already marked this order as handed over"}
		}
		return qtx.MarkOrderHandedOver(ctx, order.ID)
	case markBuyer:
		if order.BuyerReceivedAt.Valid {
			return database.Order{}, &ConflictError{Message: "you have already confirmed receipt of this order"}
		}
		return qtx.MarkOrderReceived(ctx, order.ID)
	}
	return order, nil
}

// bothSidesMarked reports whether the handshake is finished.
func bothSidesMarked(o database.Order) bool {
	return o.SellerHandedOverAt.Valid && o.BuyerReceivedAt.Valid
}

// checkOrderActor enforces the "who may travel this edge" guard.
func checkOrderActor(order database.Order, userID uuid.UUID, action orderAction) error {
	isBuyer := order.BuyerID == userID
	isSeller := order.SellerID == userID

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
	}

	return nil
}

func checkHandshakeLock(order database.Order, action orderAction) error {
	if action.blockedAfterMark &&
		(order.SellerHandedOverAt.Valid || order.BuyerReceivedAt.Valid) {
		return &ConflictError{
			Message: "cannot cancel an order once handover has started",
		}
	}
	return nil
}
