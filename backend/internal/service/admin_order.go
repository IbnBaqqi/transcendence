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
)

const maxResolutionReason = 500

type AdminOrderService struct {
	db *database.DB
}

func NewAdminOrderService(db *database.DB) *AdminOrderService {
	return &AdminOrderService{db: db}
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

	page, limit, offset, err := parsePaging(q.Page, q.Limit)
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	status := sql.NullString{String: q.Status, Valid: q.Status != ""}

	total, err := s.db.CountOrdersForAdmin(ctx, database.CountOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck,
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	rows, err := s.db.ListOrdersForAdmin(ctx, database.ListOrdersForAdminParams{
		Status: status, CreatedFrom: from, CreatedTo: to, Stuck: stuck,
		PageLimit: int32(limit), PageOffset: int32(offset),
	})
	if err != nil {
		return dtos.PaginatedAdminOrders{}, err
	}

	items := make([]dtos.AdminOrderResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, dtos.ToAdminOrderResponse(row, isStuck(row)))
	}

	return dtos.PaginatedAdminOrders{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
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

func parsePaging(rawPage, rawLimit string) (page, limit, offset int, err error) {
	page = defaultPage
	if rawPage != "" {
		p, convErr := strconv.Atoi(rawPage)
		if convErr != nil || p < 1 || p > math.MaxInt32 {
			return 0, 0, 0, &ValidationError{Message: "Page must be a positive integer"}
		}
		page = p
	}

	limit = defaultLimit
	if rawLimit != "" {
		l, convErr := strconv.Atoi(rawLimit)
		if convErr != nil || l < 1 {
			return 0, 0, 0, &ValidationError{Message: "Limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset = (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return 0, 0, 0, &ValidationError{Message: "Page is too large"}
	}

	return page, limit, offset, nil
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

func isStuck(o database.Order) bool {
	return o.Status == "confirmed" &&
		o.SellerHandedOverAt.Valid != o.BuyerReceivedAt.Valid
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

	if !isStuck(order) {
		return database.Order{}, &ConflictError{
			Message: "Only an order stuck mid-handover can be resolved by an admin",
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

	return updated, nil
}
