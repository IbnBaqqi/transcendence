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
func scrubAccount(ctx context.Context, qtx *database.Queries, userID uuid.UUID) (sql.NullString, error) {
	avatar, err := qtx.ScrubProfile(ctx, userID)
	if err != nil {
		return sql.NullString{}, err
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
			return sql.NullString{}, err
		}
	}

	if _, err := qtx.AnonymiseUser(ctx, userID); err != nil {
		return sql.NullString{}, err
	}

	return avatar, nil
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

	avatar, err := scrubAccount(ctx, qtx, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// After the commit and unable to fail it: the deletion is what the user
	// asked for, the email is a courtesy. Notify queues rather than sends.
	s.notify.Notify(context.WithoutCancel(ctx), notify.AccountDeleted(recipient, name))

	if avatar.Valid {
		if err := s.files.Delete(avatar.String); err != nil {
			slog.Error("could not delete a deleted user's avatar",
				"user_id", userID, "filename", avatar.String, "error", err)
		}
	}

	return nil
}
