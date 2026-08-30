package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/google/uuid"
)

type UserService struct {
	db    *database.DB
	files fileStore
}

func NewUserService(db *database.DB, files fileStore) *UserService {
	return &UserService{db: db, files: files}
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

	avatar, err := scrubAccount(ctx, qtx, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if avatar.Valid {
		if err := s.files.Delete(avatar.String); err != nil {
			slog.Error("could not delete a deleted user's avatar",
				"user_id", userID, "filename", avatar.String, "error", err)
		}
	}

	return nil
}
