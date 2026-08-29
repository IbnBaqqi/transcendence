package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
)

const maxActionNote = 500

type AdminUserService struct {
	db    *database.DB
	files fileStore
}

func NewAdminUserService(db *database.DB, files fileStore) *AdminUserService {
	return &AdminUserService{db: db, files: files}
}

func validateNote(note string, required bool) (string, error) {
	if !utf8.ValidString(note) || strings.ContainsRune(note, 0) {
		return "", &ValidationError{Message: "Reason must be valid UTF-8 without null bytes"}
	}

	note = sanitizeReportDetail(note)

	if utf8.RuneCountInString(note) > maxActionNote {
		return "", &ValidationError{Message: "Reason is too long"}
	}
	if required && note == "" {
		return "", &ValidationError{Message: "A reason is required"}
	}

	return note, nil
}

func guardTarget(ctx context.Context, qtx *database.Queries, adminID uuid.UUID, subject database.User) error {
	if subject.ID == adminID {
		return &ForbiddenError{Message: "You cannot do this to your own account"}
	}

	if subject.Role != auth.RoleAdmin {
		return nil
	}

	admins, err := qtx.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if admins <= 1 {
		return &ConflictError{Message: "This is the last active admin"}
	}

	return nil
}

func (s *AdminUserService) Suspend(ctx context.Context, adminID, subjectID uuid.UUID, reason string) (database.User, error) {
	reason, err := validateNote(reason, true)
	if err != nil {
		return database.User{}, err
	}
	return s.transition(ctx, adminID, subjectID, "suspended", reason)
}

func (s *AdminUserService) Reinstate(ctx context.Context, adminID, subjectID uuid.UUID, note string) (database.User, error) {
	note, err := validateNote(note, false)
	if err != nil {
		return database.User{}, err
	}
	return s.transition(ctx, adminID, subjectID, "reinstated", note)
}

func (s *AdminUserService) transition(
	ctx context.Context,
	adminID, subjectID uuid.UUID,
	action, note string,
) (database.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.User{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("admin user transition rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	subject, err := qtx.GetUserForUpdate(ctx, subjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return database.User{}, &NotFoundError{Message: "User not found"}
		}
		return database.User{}, err
	}
	if subject.DeletedAt.Valid {
		return database.User{}, &NotFoundError{Message: "User not found"}
	}

	if action == "suspended" {
		if err := guardTarget(ctx, qtx, adminID, subject); err != nil {
			return database.User{}, err
		}
	} else if subject.ID == adminID {
		return database.User{}, &ForbiddenError{Message: "You cannot do this to your own account"}
	}

	var updated database.User
	if action == "suspended" {
		updated, err = qtx.SuspendUser(ctx, database.SuspendUserParams{ID: subjectID, Reason: sql.NullString{String: note, Valid: true}})
	} else {
		updated, err = qtx.ReinstateUser(ctx, subjectID)
	}
	if errors.Is(err, sql.ErrNoRows) {
		// The query's WHERE carries the precondition, so no rows means the
		// account was already in the state being asked for.
		return database.User{}, &ConflictError{Message: "The account is already in that state"}
	}
	if err != nil {
		return database.User{}, err
	}

	if err := s.record(ctx, qtx, adminID, subjectID, action, note); err != nil {
		return database.User{}, err
	}

	if err := tx.Commit(); err != nil {
		return database.User{}, err
	}

	return updated, nil
}

func (s *AdminUserService) Delete(ctx context.Context, adminID, subjectID uuid.UUID, confirmation, reason string) error {
	reason, err := validateNote(reason, true)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("admin user deletion rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	subject, err := qtx.GetUserForUpdate(ctx, subjectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "User not found"}
		}
		return err
	}
	if subject.DeletedAt.Valid {
		return &NotFoundError{Message: "User not found"}
	}

	if err := guardTarget(ctx, qtx, adminID, subject); err != nil {
		return err
	}

	if confirmation != subject.Username {
		return &ValidationError{Message: "Type the account's username exactly to confirm"}
	}

	avatar, err := scrubAccount(ctx, qtx, subjectID)
	if err != nil {
		return err
	}

	if err := s.record(ctx, qtx, adminID, subjectID, "deleted", reason); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if avatar.Valid {
		if err := s.files.Delete(avatar.String); err != nil {
			slog.Error("could not delete a deleted user's avatar",
				"user_id", subjectID, "filename", avatar.String, "error", err)
		}
	}

	return nil
}

func (s *AdminUserService) record(ctx context.Context, qtx *database.Queries, adminID, subjectID uuid.UUID, action, note string) error {
	_, err := qtx.CreateUserAction(ctx, database.CreateUserActionParams{
		ID:          database.NewID(),
		SubjectID:   subjectID,
		ModeratorID: uuid.NullUUID{UUID: adminID, Valid: true},
		Action:      action,
		Note:        sql.NullString{String: note, Valid: note != ""},
	})
	return err
}

func (s *AdminUserService) History(ctx context.Context, subjectID uuid.UUID) ([]database.UserAction, error) {
	return s.db.ListUserActions(ctx, subjectID)
}
