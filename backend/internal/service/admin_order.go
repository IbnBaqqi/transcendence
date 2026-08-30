package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

const (
	maxResolutionReason = 500

	stuckAfter = 7 * 24 * time.Hour
)

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
	before := stuckBefore(time.Now())

	total, err := s.db.CountOrdersForAdmin(ctx, database.CountOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck, StuckBefore: before,
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	rows, err := s.db.ListOrdersForAdmin(ctx, database.ListOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck, StuckBefore: before,
		PageLimit: paging.pageLimit, PageOffset: paging.pageOffset,
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	items := make([]dtos.AdminOrderResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, dtos.ToAdminOrderResponse(row, isStuck(row, before)))
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

type paging struct {
	page       int
	limit      int
	pageLimit  int32
	pageOffset int32
}

func parsePaging(rawPage, rawLimit string) (paging, error) {
	page := defaultPage
	if rawPage != "" {
		p, convErr := strconv.Atoi(rawPage)
		if convErr != nil || p < 1 || p > math.MaxInt32 {
			return paging{}, &ValidationError{Message: "Page must be a positive integer"}
		}
		page = p
	}

	limit := defaultLimit
	if rawLimit != "" {
		l, convErr := strconv.Atoi(rawLimit)
		if convErr != nil || l < 1 {
			return paging{}, &ValidationError{Message: "Limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return paging{}, &ValidationError{Message: "Page is too large"}
	}

	return paging{
		page:       page,
		limit:      limit,
		pageLimit:  int32(limit),
		pageOffset: int32(offset),
	}, nil
}

func validateResolutionReason(reason string) (string, error) {
	if !utf8.ValidString(reason) || strings.ContainsRune(reason, 0) {
		return "", &ValidationError{Message: "Reason must be valid UTF-8 without null bytes"}
	}

	reason = strings.TrimSpace(sanitizeReportDetail(reason))

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

func stuckBefore(now time.Time) time.Time {
	return now.Add(-stuckAfter)
}

func isStuck(o database.Order, before time.Time) bool {
	if o.Status != "confirmed" || o.SellerHandedOverAt.Valid == o.BuyerReceivedAt.Valid {
		return false
	}

	marked := o.SellerHandedOverAt.Time
	if o.BuyerReceivedAt.Valid {
		marked = o.BuyerReceivedAt.Time
	}

	return marked.Before(before)
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

	if !isStuck(order, stuckBefore(time.Now())) {
		return database.Order{}, &ConflictError{
			Message: "Only an order left stuck mid-handover for 7 days can be resolved by an admin",
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
