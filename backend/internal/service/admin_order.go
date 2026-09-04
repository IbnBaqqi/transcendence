package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

const maxResolutionReason = 500

type AdminOrderService struct {
	db     *database.DB
	notify notify.Notifier
}

func NewAdminOrderService(db *database.DB, notifier notify.Notifier) *AdminOrderService {
	return &AdminOrderService{db: db, notify: notifier}
}

var adminOrderStatuses = map[string]bool{
	"pending": true, "confirmed": true, "completed": true,
	"cancelled": true, "refunded": true,
}

func (s *AdminOrderService) List(ctx context.Context, q dtos.AdminOrderQuery) (dtos.PaginatedAdminOrders, error) {
	if q.Status != "" && !adminOrderStatuses[q.Status] {
		return dtos.PaginatedAdminOrders{}, &ValidationError{
			Message: "Status must be pending, confirmed, completed, cancelled or refunded",
		}
	}

	from, err := parseDayBound(q.CreatedFrom, "created_from")
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}
	to, err := parseDayBound(q.CreatedTo, "created_to")
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}
	if from.Valid && to.Valid && !to.Time.After(from.Time) {
		return dtos.PaginatedAdminOrders{}, &ValidationError{Message: "The date range ends before it starts"}
	}

	stuck := sql.NullBool{}
	if q.Stuck != "" {
		value, parseErr := strconv.ParseBool(q.Stuck)
		if parseErr != nil {
			return dtos.PaginatedAdminOrders{}, &ValidationError{Message: "Stuck must be true or false"}
		}
		stuck = sql.NullBool{Bool: value, Valid: true}
	}

	paging, err := parsePaging(q.Page, q.Limit)
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	status := sql.NullString{String: q.Status, Valid: q.Status != ""}

	// One transaction so both queries share it: the view's stuck test is
	// relative to now(), which is the transaction timestamp, so running them
	// apart lets an order on the seven-day boundary be counted but not listed.
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("admin order listing rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	total, err := qtx.CountOrdersForAdmin(ctx, database.CountOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck,
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	rows, err := qtx.ListOrdersForAdmin(ctx, database.ListOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck,
		PageLimit: paging.pageLimit, PageOffset: paging.pageOffset,
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	if err := tx.Commit(); err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	items := make([]dtos.AdminOrderResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, dtos.ToAdminOrderResponse(orderFromAdminRow(row), row.Stuck))
	}

	return dtos.PaginatedAdminOrders{
		Items:      items,
		Total:      total,
		Page:       paging.page,
		Limit:      paging.limit,
		TotalPages: int((total + int64(paging.limit) - 1) / int64(paging.limit)),
	}, nil
}

func parseDayBound(value, field string) (sql.NullTime, error) {
	if value == "" {
		return sql.NullTime{}, nil
	}

	when, err := time.Parse(time.RFC3339, value)
	if err != nil {
		if day, dayErr := time.Parse(time.DateOnly, value); dayErr == nil {
			return sql.NullTime{Time: day, Valid: true}, nil
		}
		return sql.NullTime{}, &ValidationError{
			Message: "The " + field + " date must be RFC 3339 or YYYY-MM-DD",
		}
	}

	return sql.NullTime{Time: when, Valid: true}, nil
}

func orderFromAdminRow(row database.AdminOrder) database.Order {
	return database.Order{
		ID:                 row.ID,
		ListingID:          row.ListingID,
		BuyerID:            row.BuyerID,
		SellerID:           row.SellerID,
		Quantity:           row.Quantity,
		UnitPrice:          row.UnitPrice,
		TotalPrice:         row.TotalPrice,
		Status:             row.Status,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		SellerHandedOverAt: row.SellerHandedOverAt,
		BuyerReceivedAt:    row.BuyerReceivedAt,
		ListingTitle:       row.ListingTitle,
	}
}

func validateResolutionReason(reason string) (string, error) {
	if !utf8.ValidString(reason) || strings.ContainsRune(reason, 0) {
		return "", &ValidationError{Message: "Reason must be valid UTF-8 without null bytes"}
	}

	reason = strings.TrimSpace(sanitizeFreeText(reason))

	if reason == "" {
		return "", &ValidationError{Message: "A reason is required"}
	}
	if utf8.RuneCountInString(reason) > maxResolutionReason {
		return "", &ValidationError{Message: "Reason is too long"}
	}

	return reason, nil
}

type orderOutcome struct {
	status        string
	restoresStock bool
}

var adminOutcomes = map[string]orderOutcome{
	"completed": {status: "completed"},
	"cancelled": {status: "cancelled", restoresStock: true},
	"refunded":  {status: "refunded", restoresStock: true},
}

// ListEvents is OrderService.ListEvents without the membership check, which is
// the whole point: an admin is never a party to the order they are judging.
// The order still has to exist, or a mistyped id would answer 200 with an
// empty history and read as "nothing ever happened".
//
// Same query and same response shape as the parties get, so one schema serves
// both - an admin reading a different history from the people it is about is
// the failure this endpoint exists to avoid.
func (s *AdminOrderService) ListEvents(ctx context.Context, orderID uuid.UUID) ([]database.OrderEvent, error) {
	if _, err := s.db.GetOrder(ctx, orderID); err != nil {
		return nil, &NotFoundError{Message: "Order not found"}
	}

	return s.db.ListOrderEvents(ctx, orderID)
}

func (s *AdminOrderService) Resolve(
	ctx context.Context,
	adminID, orderID uuid.UUID,
	outcome, reason string,
) (database.Order, error) {
	action, ok := adminOutcomes[outcome]
	if !ok {
		return database.Order{}, &ValidationError{
			Message: "Outcome must be completed, cancelled or refunded",
		}
	}

	reason, err := validateResolutionReason(reason)
	if err != nil {
		return database.Order{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.Order{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("admin order resolution rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	order, err := qtx.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.Order{}, &NotFoundError{Message: "Order not found"}
		}
		return database.Order{}, err
	}

	resolvable, err := qtx.GetOrderResolvability(ctx, orderID)
	if err != nil {
		return database.Order{}, err
	}
	if !resolvable.Stuck {
		return database.Order{}, &ConflictError{
			Message: "Only an order left stuck mid-handover for 7 days, or one whose buyer and seller have both deleted their accounts, can be resolved by an admin",
		}
	}
	if resolvable.Stranded && action.status != "cancelled" {
		return database.Order{}, &ConflictError{
			Message: "An order neither party ever acted on can only be resolved as cancelled",
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
		ID:     orderID,
		Status: action.status,
	})
	if err != nil {
		return database.Order{}, err
	}

	if err := recordEvent(ctx, qtx, orderID, adminID,
		sql.NullString{String: order.Status, Valid: true}, action.status, reason); err != nil {
		return database.Order{}, err
	}

	// Both parties, same as the emails below: an admin resolving a dispute is
	// news to whichever of them did not ask for it as much as to the one who did.
	for _, recipient := range []uuid.UUID{order.BuyerID, order.SellerID} {
		if err := recordOrderNotification(ctx, qtx, recipient, notifyKindOrderResolved, updated); err != nil {
			return database.Order{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return database.Order{}, err
	}

	for _, recipient := range []uuid.UUID{order.BuyerID, order.SellerID} {
		notifyUser(ctx, s.db.Queries, s.notify, recipient,
			func(email, _ string) notify.Message {
				return notify.OrderResolved(email, order.ListingTitle, action.status)
			})
	}

	return updated, nil
}
