package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/auth"
	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
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

	// An already-suspended admin is not one of the active admins CountAdmins
	// counts, so guarding them would block the suspend-then-delete path over
	// a shortage they are not part of.
	if subject.Role != auth.RoleAdmin || subject.SuspendedAt.Valid {
		return nil
	}

	if err := qtx.LockAdminRoster(ctx); err != nil {
		return err
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

var adminUserStatuses = map[string]bool{"active": true, "suspended": true, "deleted": true}

func (s *AdminUserService) List(ctx context.Context, q dtos.AdminUserQuery) (dtos.PaginatedAdminUsers, error) {
	if q.Role != "" && q.Role != auth.RoleUser && q.Role != auth.RoleAdmin {
		return dtos.PaginatedAdminUsers{}, &ValidationError{Message: "Role must be USER or ADMIN"}
	}
	if q.Status != "" && !adminUserStatuses[q.Status] {
		return dtos.PaginatedAdminUsers{}, &ValidationError{
			Message: "Status must be active, suspended or deleted",
		}
	}

	page := defaultPage
	if q.Page != "" {
		p, err := strconv.Atoi(q.Page)
		if err != nil || p < 1 || p > math.MaxInt32 {
			return dtos.PaginatedAdminUsers{}, &ValidationError{Message: "Page must be a positive integer"}
		}
		page = p
	}

	limit := defaultLimit
	if q.Limit != "" {
		l, err := strconv.Atoi(q.Limit)
		if err != nil || l < 1 {
			return dtos.PaginatedAdminUsers{}, &ValidationError{Message: "Limit must be a positive integer"}
		}
		limit = l
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset := (page - 1) * limit
	if offset < 0 || offset > math.MaxInt32 {
		return dtos.PaginatedAdminUsers{}, &ValidationError{Message: "Page is too large"}
	}

	role := sql.NullString{String: q.Role, Valid: q.Role != ""}
	status := sql.NullString{String: q.Status, Valid: q.Status != ""}

	total, err := s.db.CountUsersForAdmin(ctx, database.CountUsersForAdminParams{Role: role, Status: status})
	if err != nil {
		return dtos.PaginatedAdminUsers{}, err
	}

	rows, err := s.db.ListUsersForAdmin(ctx, database.ListUsersForAdminParams{
		Role:       role,
		Status:     status,
		PageLimit:  int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return dtos.PaginatedAdminUsers{}, err
	}

	return dtos.PaginatedAdminUsers{
		Items:      dtos.ToAdminUserResponses(rows),
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	}, nil
}

func (s *AdminUserService) History(ctx context.Context, subjectID uuid.UUID) ([]database.UserAction, error) {
	return s.db.ListUserActions(ctx, subjectID)
}
