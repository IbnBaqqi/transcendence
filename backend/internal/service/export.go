package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
	"github.com/IbnBaqqi/transcendence/internal/notify"
)

// ExportData gathers everything the API already tells this account about
// itself into one document.
//
// Two things are deliberately absent, and both are absent everywhere else in
// the API for the same reason:
//
//   - who has blocked THIS account. A block is symmetric in its effects
//     precisely so neither side learns who acted (see block.go), and an export
//     that named them would be the disclosure channel the API refuses to be.
//   - moderator notes from user_actions. They are written for admins about a
//     person, not for the person, and no endpoint shows them.
//
// The other party's messages and the authors of reviews received ARE included:
// both are readable in the app today, so exporting them discloses nothing new,
// and leaving them out would produce half a conversation.
func (s *UserService) ExportData(ctx context.Context, userID uuid.UUID) (dtos.DataExport, error) {
	q := s.db.Queries

	user, err := q.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dtos.DataExport{}, &NotFoundError{Message: "User not found"}
		}
		return dtos.DataExport{}, err
	}
	if user.DeletedAt.Valid {
		return dtos.DataExport{}, &NotFoundError{Message: "User not found"}
	}

	profile, err := q.GetProfile(ctx, userID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return dtos.DataExport{}, err
	}

	// An account may have no address row at all, which is not an error - the
	// profile response carries a null location for exactly that case.
	var location sql.NullString
	address, err := q.GetAddress(ctx, userID)
	if err == nil {
		location = address.Location
	} else if !errors.Is(err, sql.ErrNoRows) {
		return dtos.DataExport{}, err
	}

	providers, err := q.ListProvidersForUser(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	listings, err := q.ListListingsForExport(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	orders, err := q.ListOrdersForUser(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	saved, err := q.ListSavedListings(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	conversations, err := q.ListConversationsForUser(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	messages, err := q.ListMessagesForExport(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	written, err := q.ListReviewsWrittenBy(ctx, uuid.NullUUID{UUID: userID, Valid: true})
	if err != nil {
		return dtos.DataExport{}, err
	}

	received, err := q.ListReviewsAboutUserForExport(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	// ViewerID and SubjectID are the same account here: this is the caller
	// reading their own graph, which is what the export is.
	following, err := q.ListFollowing(ctx, database.ListFollowingParams{ViewerID: userID, SubjectID: userID})
	if err != nil {
		return dtos.DataExport{}, err
	}

	followers, err := q.ListFollowers(ctx, database.ListFollowersParams{ViewerID: userID, SubjectID: userID})
	if err != nil {
		return dtos.DataExport{}, err
	}

	blocks, err := q.ListBlocks(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	notifications, err := q.ListNotificationsForExport(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	keys, err := q.ListKeysForUser(ctx, userID)
	if err != nil {
		return dtos.DataExport{}, err
	}

	at := time.Now().UTC()

	// Confirms that it happened; does not carry the file. Mailing someone's
	// full record would put it in an inbox and in whatever relay handled it,
	// which is the opposite of what this feature is for. Queued, so it cannot
	// fail the download the caller is waiting on.
	s.notify.Notify(context.WithoutCancel(ctx), notify.DataExported(user.Email, user.Username, at))

	return dtos.DataExport{
		ExportedAt:      at,
		Account:         dtos.ToOwnProfileResponse(user, profile, location),
		Providers:       providers,
		Listings:        dtos.ToListingResponses(listings),
		Orders:          dtos.ToExportOrders(orders),
		SavedListings:   dtos.ToListingResponses(saved),
		Conversations:   dtos.ToConversationListItems(conversations, userID),
		Messages:        dtos.ToMessageResponses(messages),
		ReviewsWritten:  dtos.ToExportReviews(written),
		ReviewsReceived: dtos.ToExportReviews(received),
		Following:       dtos.ToFollowingResponses(following),
		Followers:       dtos.ToFollowerResponses(followers),
		Blocks:          dtos.ToBlockedUserResponses(blocks),
		Notifications:   dtos.ToNotificationResponses(notifications),
		APIKeys:         dtos.ToAPIKeyResponses(keys),
	}, nil
}
