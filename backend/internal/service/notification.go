package service

import (
	"context"
	"database/sql"

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

	// The order lifecycle's other two ends. The buyer was told when their
	// order was cancelled and not when it was accepted or finished, which is
	// the half of the handshake they actually wait on.
	notifyKindOrderConfirmed = "order_confirmed"
	notifyKindOrderCompleted = "order_completed"

	notifyKindReviewReceived      = "review_received"
	notifyKindNewFollower         = "new_follower"
	notifyKindListingRemoved      = "listing_removed"
	notifyKindSavedListingGone    = "saved_listing_gone"
	notifyKindSavedListingDeleted = "saved_listing_deleted"
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
		ListingTitle: sql.NullString{String: order.ListingTitle, Valid: true},
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
		ListingTitle:   sql.NullString{String: conv.ListingTitle, Valid: true},
		ConversationID: uuid.NullUUID{UUID: conv.ID, Valid: true},
	})
}

func recordActorNotification(
	ctx context.Context,
	qtx *database.Queries,
	userID uuid.UUID,
	kind string,
	actorID uuid.UUID,
	listingTitle sql.NullString,
) error {
	return qtx.CreateNotification(ctx, database.CreateNotificationParams{
		ID:           database.NewID(),
		UserID:       userID,
		Kind:         kind,
		ListingTitle: listingTitle,
		ActorID:      uuid.NullUUID{UUID: actorID, Valid: true},
	})
}

func recordListingNotification(
	ctx context.Context,
	qtx *database.Queries,
	userID uuid.UUID,
	kind string,
	listing database.Listing,
) error {
	return qtx.CreateNotification(ctx, database.CreateNotificationParams{
		ID:           database.NewID(),
		UserID:       userID,
		Kind:         kind,
		ListingTitle: sql.NullString{String: listing.Title, Valid: true},
		ListingID:    uuid.NullUUID{UUID: listing.ID, Valid: true},
	})
}

func notifySavers(
	ctx context.Context,
	qtx *database.Queries,
	listing database.Listing,
	except uuid.UUID,
	kind string,
) error {
	savers, err := qtx.ListSaversOfListing(ctx, database.ListSaversOfListingParams{
		ListingID:  listing.ID,
		ExceptUser: except,
	})
	if err != nil {
		return err
	}

	for _, saver := range savers {
		// A deleted listing takes its notifications with it: listing_id is a
		// foreign key that cascades, so the row has to point at the seller,
		// who survives. Switching this to the listing loses every row silently.
		if kind == notifyKindSavedListingDeleted {
			err = recordActorNotification(ctx, qtx, saver, kind, listing.SellerID,
				sql.NullString{String: listing.Title, Valid: true})
		} else {
			err = recordListingNotification(ctx, qtx, saver, kind, listing)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Restocking is what makes an existing saved_listing_gone false, so the rows
// go when the stock comes back. Without this a buy-and-cancel cycle leaves a
// "sold out" notice on a listing that is in stock, and repeating it fills the
// reader's 30-row inbox with copies until nothing else is left in it.
func restock(ctx context.Context, qtx *database.Queries, listingID uuid.UUID, quantity int32) error {
	if _, err := qtx.IncrementListingQuantity(ctx, database.IncrementListingQuantityParams{
		ID:       listingID,
		Quantity: quantity,
	}); err != nil {
		return err
	}

	return qtx.DeleteNotificationsForListingKind(ctx, database.DeleteNotificationsForListingKindParams{
		ListingID: uuid.NullUUID{UUID: listingID, Valid: true},
		Kind:      notifyKindSavedListingGone,
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
