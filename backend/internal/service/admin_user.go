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

	note = sanitizeFreeText(note)

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
		if err == nil {
			if revokeErr := qtx.RevokeSessionsForUser(ctx, subjectID); revokeErr != nil {
				return database.User{}, revokeErr
			}
		}
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

// SetRole promotes or demotes an account. The role is read from the database
// on every request that needs it, so it takes effect on the caller's NEXT
// request rather than when their token expires - which is what makes a
// demotion mean anything.
func (s *AdminUserService) SetRole(ctx context.Context, adminID, subjectID uuid.UUID, role, note string) (database.User, error) {
	if role != auth.RoleUser && role != auth.RoleAdmin {
		return database.User{}, &ValidationError{Message: "Role must be USER or ADMIN"}
	}
	note, err := validateNote(note, false)
	if err != nil {
		return database.User{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return database.User{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("admin role change rollback failed", "error", err)
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

	if subject.ID == adminID {
		return database.User{}, &ForbiddenError{Message: "You cannot do this to your own account"}
	}

	// Only a demotion can empty the roster, and guardTarget is what takes the
	// lock and counts INSIDE this transaction. Read the count outside it and
	// two admins demoting each other both see two admins, both write, and
	// nobody is left who can grant the role back.
	if role == auth.RoleUser {
		if err := guardTarget(ctx, qtx, adminID, subject); err != nil {
			return database.User{}, err
		}
	}

	updated, err := qtx.SetUserRole(ctx, database.SetUserRoleParams{ID: subjectID, Role: role})
	if errors.Is(err, sql.ErrNoRows) {
		// The query's WHERE carries the precondition, so no rows means the
		// account already has this role. That is not an error and must not
		// write history: a promotion that did not happen has no business in
		// GET /admin/users/{id}/history.
		if err := tx.Commit(); err != nil {
			return database.User{}, err
		}
		return subject, nil
	}
	if err != nil {
		return database.User{}, err
	}

	action := "promoted"
	if role == auth.RoleUser {
		action = "demoted"
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

	scrubbed, err := scrubAccount(ctx, qtx, subjectID)
	if err != nil {
		return err
	}

	if err := s.record(ctx, qtx, adminID, subjectID, "deleted", reason); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	deleteScrubbedFiles(s.files, subjectID, scrubbed)

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

	paging, err := parsePaging(q.Page, q.Limit)
	if err != nil {
		return dtos.PaginatedAdminUsers{}, err
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
		PageLimit:  paging.pageLimit,
		PageOffset: paging.pageOffset,
	})
	if err != nil {
		return dtos.PaginatedAdminUsers{}, err
	}

	return dtos.PaginatedAdminUsers{
		Items:      dtos.ToAdminUserResponses(rows),
		Total:      total,
		Page:       paging.page,
		Limit:      paging.limit,
		TotalPages: int((total + int64(paging.limit) - 1) / int64(paging.limit)),
	}, nil
}

func (s *AdminUserService) History(ctx context.Context, subjectID uuid.UUID) ([]database.UserAction, error) {
	// Deliberately not the DeletedAt check the transitions make: deletion here
	// anonymises, and the trail contains the deletion itself. 404ing a deleted
	// account would hide the record of its own deletion.
	if _, err := s.db.GetUser(ctx, subjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &NotFoundError{Message: "User not found"}
		}
		return nil, err
	}

	return s.db.ListUserActions(ctx, subjectID)
}
