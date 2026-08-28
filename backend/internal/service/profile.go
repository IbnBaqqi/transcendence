package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
	"github.com/IbnBaqqi/transcendence/internal/dtos"
)

const (
	maxNameLength     = 150
	maxBioLength      = 1000
	maxPhoneLength    = 15
	maxLocationLength = 100
	earliestBirthYear = 1900
)

type ProfileDetail struct {
	User     database.User
	Profile  database.Profile
	Location sql.NullString
	Rating   database.SellerRatingRow
}

type ProfileService struct {
	db    *database.DB
	files fileStore
}

func NewProfileService(db *database.DB, files fileStore) *ProfileService {
	return &ProfileService{db: db, files: files}
}

func (s *ProfileService) Get(ctx context.Context, userID uuid.UUID) (ProfileDetail, error) {
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileDetail{}, &NotFoundError{Message: "User not found"}
		}
		return ProfileDetail{}, err
	}

	if user.DeletedAt.Valid {
		return ProfileDetail{}, &NotFoundError{Message: "User not found"}
	}

	profile, err := s.db.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileDetail{}, &NotFoundError{Message: "Profile not found"}
		}
		return ProfileDetail{}, err
	}

	location, err := s.location(ctx, s.db.Queries, userID)
	if err != nil {
		return ProfileDetail{}, err
	}

	rating, err := s.db.SellerRating(ctx, userID)
	if err != nil {
		return ProfileDetail{}, err
	}

	return ProfileDetail{User: user, Profile: profile, Location: location, Rating: rating}, nil
}

func (s *ProfileService) Update(ctx context.Context, userID uuid.UUID, input dtos.UpdateProfileInput) (ProfileDetail, error) {
	if err := validateProfileInput(input); err != nil {
		return ProfileDetail{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProfileDetail{}, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("profile update transaction rollback failed", "error", err)
		}
	}()

	qtx := s.db.Queries.WithTx(tx.Tx)

	current, err := qtx.GetProfileForUpdate(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileDetail{}, &NotFoundError{Message: "Profile not found"}
		}
		return ProfileDetail{}, err
	}

	dob, err := mergeDate(current.DateOfBirth, input.DateOfBirth)
	if err != nil {
		return ProfileDetail{}, err
	}

	profile, err := qtx.UpdateProfile(ctx, database.UpdateProfileParams{
		ID:          userID,
		Firstname:   mergeString(current.Firstname, input.Firstname),
		Lastname:    mergeString(current.Lastname, input.Lastname),
		Bio:         mergeString(current.Bio, input.Bio),
		PhoneNumber: mergeString(current.PhoneNumber, input.PhoneNumber),
		DateOfBirth: dob,
	})
	if err != nil {
		return ProfileDetail{}, err
	}

	location, err := s.location(ctx, qtx, userID)
	if err != nil {
		return ProfileDetail{}, err
	}
	if input.Location.Set {
		address, err := qtx.UpsertAddress(ctx, database.UpsertAddressParams{
			ID:       database.NewID(),
			UserID:   userID,
			Location: mergeString(location, input.Location),
		})
		if err != nil {
			return ProfileDetail{}, err
		}
		location = address.Location
	}

	user, err := qtx.GetUser(ctx, userID)
	if err != nil {
		return ProfileDetail{}, err
	}

	if err := tx.Commit(); err != nil {
		return ProfileDetail{}, err
	}

	return ProfileDetail{User: user, Profile: profile, Location: location}, nil
}

func (s *ProfileService) location(ctx context.Context, q *database.Queries, userID uuid.UUID) (sql.NullString, error) {
	address, err := q.GetAddress(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.NullString{}, nil
		}
		return sql.NullString{}, err
	}
	return address.Location, nil
}

func mergeString(current sql.NullString, in dtos.OptionalString) sql.NullString {
	if !in.Set {
		return current
	}
	if in.Value == nil {
		return sql.NullString{}
	}
	trimmed := strings.TrimSpace(*in.Value)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func mergeDate(current sql.NullTime, in dtos.OptionalString) (sql.NullTime, error) {
	if !in.Set {
		return current, nil
	}
	if in.Value == nil {
		return sql.NullTime{}, nil
	}
	trimmed := strings.TrimSpace(*in.Value)
	if trimmed == "" {
		return sql.NullTime{}, nil
	}

	parsed, err := time.Parse(dtos.DateLayout, trimmed)
	if err != nil {
		return sql.NullTime{}, &ValidationError{Message: "Date of birth must look like 2001-05-14"}
	}
	if parsed.After(time.Now()) {
		return sql.NullTime{}, &ValidationError{Message: "Date of birth cannot be in the future"}
	}
	if parsed.Year() < earliestBirthYear {
		return sql.NullTime{}, &ValidationError{Message: "Date of birth is implausibly early"}
	}
	return sql.NullTime{Time: parsed, Valid: true}, nil
}

func validateProfileInput(input dtos.UpdateProfileInput) error {
	limits := []struct {
		field string
		value dtos.OptionalString
		max   int
	}{
		// Display names, not JSON field names: these are read by a person, so
		// "Phone number is too long" rather than "phone_number is too long".
		{"First name", input.Firstname, maxNameLength},
		{"Last name", input.Lastname, maxNameLength},
		{"Bio", input.Bio, maxBioLength},
		{"Phone number", input.PhoneNumber, maxPhoneLength},
		{"Location", input.Location, maxLocationLength},
	}

	for _, l := range limits {
		if !l.value.Set || l.value.Value == nil {
			continue
		}
		value := strings.TrimSpace(*l.value.Value)

		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return &ValidationError{Message: l.field + " must be valid UTF-8 without null bytes"}
		}
		if len(value) > l.max {
			return &ValidationError{Message: l.field + " is too long"}
		}
	}
	return nil
}

func (s *ProfileService) SetAvatar(
	ctx context.Context,
	userID uuid.UUID,
	r io.Reader,
	ext string,
) (string, error) {
	filename, err := s.files.Save(r, ext)
	if err != nil {
		return "", err
	}

	previous, err := s.db.SetAvatar(ctx, database.SetAvatarParams{
		ID:             userID,
		AvatarFilename: sql.NullString{String: filename, Valid: true},
	})
	if err != nil {
		if delErr := s.files.Delete(filename); delErr != nil {
			slog.Error("orphaned upload: file written but avatar update failed",
				"filename", filename, "error", delErr)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return "", &NotFoundError{Message: "Profile not found"}
		}
		return "", err
	}

	s.deleteReplaced(previous)

	return filename, nil
}

func (s *ProfileService) RemoveAvatar(ctx context.Context, userID uuid.UUID) error {
	previous, err := s.db.ClearAvatar(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &NotFoundError{Message: "Profile not found"}
		}
		return err
	}

	s.deleteReplaced(previous)
	return nil
}

// deleteReplaced is cleanup, not part of the request: the row is already
// correct, so a failure here is a stray file and a log line, not a 500.
func (s *ProfileService) deleteReplaced(previous sql.NullString) {
	if !previous.Valid {
		return
	}
	if err := s.files.Delete(previous.String); err != nil {
		slog.Error("replaced avatar left on disk",
			"filename", previous.String, "error", err)
	}
}
