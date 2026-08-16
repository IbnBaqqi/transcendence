package service

import (
	"context"
	"database/sql"
	"errors"
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

// ProfileDetail is the three rows a profile is spread across.
type ProfileDetail struct {
	User     database.User
	Profile  database.Profile
	Location sql.NullString
}

// Takes *database.DB, not *database.Queries, because Update needs BeginTx.
type ProfileService struct {
	db *database.DB
}

func NewProfileService(db *database.DB) *ProfileService {
	return &ProfileService{db: db}
}

// Get loads one user's profile.
func (s *ProfileService) Get(ctx context.Context, userID uuid.UUID) (ProfileDetail, error) {
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileDetail{}, &NotFoundError{Message: "user not found"}
		}
		return ProfileDetail{}, err
	}

	profile, err := s.db.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ProfileDetail{}, &NotFoundError{Message: "profile not found"}
		}
		return ProfileDetail{}, err
	}

	location, err := s.location(ctx, s.db.Queries, userID)
	if err != nil {
		return ProfileDetail{}, err
	}

	return ProfileDetail{User: user, Profile: profile, Location: location}, nil
}

// Update applies a PATCH. Fields left out of the body keep their current
// value; fields sent as "" are cleared.
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
			return ProfileDetail{}, &NotFoundError{Message: "profile not found"}
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
	if input.Location != nil {
		address, err := qtx.UpsertAddress(ctx, database.UpsertAddressParams{
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

// location returns an invalid NullString when the user has no address row,
// which is the normal case rather than an error.
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

// mergeString is the PATCH rule in one place: nil keeps what is there, a
// value replaces it, and an empty value clears the column to NULL.
func mergeString(current sql.NullString, in *string) sql.NullString {
	if in == nil {
		return current
	}
	trimmed := strings.TrimSpace(*in)
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func mergeDate(current sql.NullTime, in *string) (sql.NullTime, error) {
	if in == nil {
		return current, nil
	}
	trimmed := strings.TrimSpace(*in)
	if trimmed == "" {
		return sql.NullTime{}, nil
	}

	parsed, err := time.Parse(dtos.DateLayout, trimmed)
	if err != nil {
		return sql.NullTime{}, &ValidationError{Message: "date_of_birth must look like 2001-05-14"}
	}
	if parsed.After(time.Now()) {
		return sql.NullTime{}, &ValidationError{Message: "date_of_birth cannot be in the future"}
	}
	if parsed.Year() < earliestBirthYear {
		return sql.NullTime{}, &ValidationError{Message: "date_of_birth is implausibly early"}
	}
	return sql.NullTime{Time: parsed, Valid: true}, nil
}

func validateProfileInput(input dtos.UpdateProfileInput) error {
	limits := []struct {
		field string
		value *string
		max   int
	}{
		{"firstname", input.Firstname, maxNameLength},
		{"lastname", input.Lastname, maxNameLength},
		{"bio", input.Bio, maxBioLength},
		{"phone_number", input.PhoneNumber, maxPhoneLength},
		{"location", input.Location, maxLocationLength},
	}

	for _, l := range limits {
		if l.value == nil {
			continue
		}
		value := strings.TrimSpace(*l.value)

		if !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return &ValidationError{Message: l.field + " must be valid UTF-8 without null bytes"}
		}
		if len(value) > l.max {
			return &ValidationError{Message: l.field + " is too long"}
		}
	}
	return nil
}
