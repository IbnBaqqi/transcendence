package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// The kinds the notifications_kind_check constraint accepts. Adding one means
// adding it there too, or the insert fails at runtime.
const (
	notifyKindOrderPlaced     = "order_placed"
	notifyKindOrderHandedOver = "order_handed_over"
	notifyKindOrderCancelled  = "order_cancelled"
	notifyKindOrderResolved   = "order_resolved"
	notifyKindChatRequest     = "chat_request"
)

// Written with qtx inside the caller's transaction, unlike the email beside it:
// an email cannot be unsent, so it waits until the change is certain, while a
// notification is a row in the same database and belongs with the rest of them.
// Sent after the commit it would go missing whenever the process died in
// between, for a change that definitely happened.
func recordOrderNotification(
	ctx context.Context,
	qtx *database.Queries,
	userID uuid.UUID,
	kind string,
	order database.Order,
) error {
	return qtx.CreateNotification(ctx, database.CreateNotificationParams{
		ID:           database.NewID(),
		UserID:       userID,
		Kind:         kind,
		ListingTitle: order.ListingTitle,
		OrderID:      uuid.NullUUID{UUID: order.ID, Valid: true},
	})
}

func recordChatNotification(
	ctx context.Context,
	qtx *database.Queries,
	userID uuid.UUID,
	conv database.Conversation,
) error {
	return qtx.CreateNotification(ctx, database.CreateNotificationParams{
		ID:             database.NewID(),
		UserID:         userID,
		Kind:           notifyKindChatRequest,
		ListingTitle:   conv.ListingTitle,
		ConversationID: uuid.NullUUID{UUID: conv.ID, Valid: true},
	})
}

// The centre shows what happened lately, not an archive: a cap here means the
// endpoint never has to explain itself, and there is no paging to design until
// somebody asks for one.
const recentNotifications = 30

type NotificationService struct {
	db *database.Queries
}

func NewNotificationService(db *database.Queries) *NotificationService {
	return &NotificationService{db: db}
}

func (s *NotificationService) List(ctx context.Context, userID uuid.UUID) ([]database.Notification, error) {
	return s.db.ListNotifications(ctx, database.ListNotificationsParams{
		UserID: userID,
		Limit:  recentNotifications,
	})
}

// Returns how many were still unread, which is what the caller just cleared.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.db.MarkNotificationsRead(ctx, userID)
}
