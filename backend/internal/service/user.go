package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/notify"
	"github.com/google/uuid"
)

type UserService struct {
	db     *database.DB
	files  fileStore
	notify notify.Notifier
}

func NewUserService(db *database.DB, files fileStore, notifier notify.Notifier) *UserService {
	return &UserService{db: db, files: files, notify: notifier}
}

func (s *UserService) Get(ctx context.Context, userID uuid.UUID) (database.User, error) {
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.User{}, &NotFoundError{Message: "User not found"}
		}
		return database.User{}, err
	}
	return user, nil
}

func (s *UserService) SetShowOnlineStatus(ctx context.Context, userID uuid.UUID, show bool) (database.User, error) {
	user, err := s.db.UpdateShowOnlineStatus(ctx, database.UpdateShowOnlineStatusParams{
		ID:               userID,
		ShowOnlineStatus: show,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.User{}, &NotFoundError{Message: "User not found"}
		}
		return database.User{}, err
	}
	return user, nil
}

// scrubAccount is shared by self-deletion and admin deletion on purpose: an
// account has to end up in the same state either way, and two copies would
// drift into scrubbing different things.
// scrubbedFiles is what the scrub orphaned on disk. Unlinked by the caller
// after the commit: doing it inside the transaction would leave the files gone
// if it rolled back, and a missing file cannot be put back.
type scrubbedFiles struct {
	Avatar        sql.NullString
	ListingImages []string
}

func scrubAccount(ctx context.Context, qtx *database.Queries, userID uuid.UUID) (scrubbedFiles, error) {
	avatar, err := qtx.ScrubProfile(ctx, userID)
	if err != nil {
		return scrubbedFiles{}, err
	}

	// Cancel before deleting the listings, and delete the listings before the
	// anonymise: pressing delete ends everything this account owns, then ends
	// the account. The order inside that is not arbitrary - cancelling restocks,
	// and restocking a listing about to be deleted is a wasted write, while the
	// reverse order would cost a departing buyer's seller their stock.
	if err := cancelOrdersOnDeparture(ctx, qtx, userID); err != nil {
		return scrubbedFiles{}, err
	}

	images, err := qtx.ListSellerImageFilenames(ctx, userID)
	if err != nil {
		return scrubbedFiles{}, err
	}
	if err := qtx.DeleteListingsForSeller(ctx, userID); err != nil {
		return scrubbedFiles{}, err
	}

	for _, step := range []func(context.Context, uuid.UUID) error{
		qtx.DeleteAddressesForUser,
		qtx.DeleteFollowsForUser,
		qtx.DeleteBlocksForUser,
		qtx.DeleteSavedForUser,
		qtx.DeleteIdentitiesForUser,
		qtx.InvalidateResetTokensForUser,
		qtx.DetachReporter,
		qtx.DetachModerator,
		qtx.DetachEventActor,
		qtx.DetachReviewer,
		qtx.DetachUserActionModerator,
		qtx.RevokeSessionsForUser,
		qtx.RevokeKeysForUser,
	} {
		if err := step(ctx, userID); err != nil {
			return scrubbedFiles{}, err
		}
	}

	if _, err := qtx.AnonymiseUser(ctx, userID); err != nil {
		return scrubbedFiles{}, err
	}

	return scrubbedFiles{Avatar: avatar, ListingImages: images}, nil
}

// DeleteAccount anonymises the row instead of deleting it. Do not "simplify"
// this to a DELETE: orders references users with ON DELETE RESTRICT on both
// sides, so it would fail outright for anyone who has traded, and messages
// cascade from conversations, so it would destroy the other party's copy of a
// shared thread.
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID, confirmation string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("account deletion transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	user, err := qtx.GetUserForUpdate(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "User not found"}
		}
		return err
	}
	if user.DeletedAt.Valid {
		return &NotFoundError{Message: "User not found"}
	}

	if confirmation != user.Username {
		return &ValidationError{Message: "Type your username exactly to confirm"}
	}

	// Read before the scrub, used after the commit. scrubAccount rewrites
	// users.email to deleted-<id>@example.invalid, so the usual "look the
	// recipient up afterwards" path would address this to nobody.
	recipient, name := user.Email, user.Username

	scrubbed, err := scrubAccount(ctx, qtx, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// After the commit and unable to fail it: the deletion is what the user
	// asked for, the email is a courtesy. Notify queues rather than sends.
	s.notify.Notify(context.WithoutCancel(ctx), notify.AccountDeleted(recipient, name))

	deleteScrubbedFiles(s.files, userID, scrubbed)

	return nil
}

// cancelOrdersOnDeparture ends every sale this account still has in flight.
// Without it the counterparty sits at pending waiting on a handover from a
// person who no longer exists.
//
// This is what makes admin_orders.stranded unreachable from here: that needs
// both parties gone with the order still open, and the first departure now
// closes it. The condition still describes rows written before this change.
//
// The same three steps a normal cancellation takes, through the same helpers:
// the status, the stock, and telling the other party. Reusing them is what
// keeps this from becoming a second, quieter kind of cancellation.
func cancelOrdersOnDeparture(ctx context.Context, qtx *database.Queries, userID uuid.UUID) error {
	orders, err := qtx.ListOrdersToCancelOnDeparture(ctx, userID)
	if err != nil {
		return err
	}

	for _, order := range orders {
		updated, err := qtx.UpdateOrderStatus(ctx, database.UpdateOrderStatusParams{
			ID:     order.ID,
			Status: "cancelled",
		})
		if err != nil {
			return err
		}

		// Valid: #264 made listing_id nullable, so an order outlives the
		// listing it was placed on - which is what lets the listings below be
		// deleted without costing the buyer their receipt.
		if order.ListingID.Valid {
			if err := restock(ctx, qtx, order.ListingID.UUID, order.Quantity); err != nil {
				return err
			}
		}

		other := order.SellerID
		if other == userID {
			other = order.BuyerID
		}
		if err := recordOrderNotification(ctx, qtx, other, notifyKindOrderCancelled, updated); err != nil {
			return err
		}
	}

	return nil
}

// deleteScrubbedFiles unlinks what the scrub orphaned - the avatar and every
// image on the listings it deleted. Best effort and after the commit: a file
// left behind is waste, and a failed unlink is not worth undoing a deletion the
// account already asked for.
func deleteScrubbedFiles(files fileStore, userID uuid.UUID, scrubbed scrubbedFiles) {
	names := make([]string, 0, len(scrubbed.ListingImages)+1)
	names = append(names, scrubbed.ListingImages...)
	if scrubbed.Avatar.Valid {
		names = append(names, scrubbed.Avatar.String)
	}

	for _, name := range names {
		if err := files.Delete(name); err != nil {
			slog.Error("could not delete a departed account's file",
				"user_id", userID, "filename", name, "error", err)
		}
	}
}
