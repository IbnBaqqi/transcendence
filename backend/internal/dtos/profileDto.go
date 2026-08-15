package dtos

import (
	"database/sql"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

// The date_of_birth column is a DATE, not a timestamp.
const DateLayout = "2006-01-02"

// UpdateProfileInput is the PATCH body.
type UpdateProfileInput struct {
	Firstname   *string `json:"firstname"`
	Lastname    *string `json:"lastname"`
	Bio         *string `json:"bio"`
	PhoneNumber *string `json:"phone_number"`
	DateOfBirth *string `json:"date_of_birth"`
	Location    *string `json:"location"`
}

// OwnProfileResponse is what you get about YOURSELF: everything, including
// the fields nobody else may see.
type OwnProfileResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Firstname   *string   `json:"firstname"`
	Lastname    *string   `json:"lastname"`
	Bio         *string   `json:"bio"`
	PhoneNumber *string   `json:"phone_number"`
	DateOfBirth *string   `json:"date_of_birth"`
	Location    *string   `json:"location"`
}

// PublicProfileResponse is what everyone else gets. Email, phone number and
// date of birth are not blanked here - they do not EXIST here, so no future
// handler can leak them by forgetting a step.
type PublicProfileResponse struct {
	ID        uuid.UUID        `json:"id"`
	Username  string           `json:"username"`
	Firstname *string          `json:"firstname"`
	Lastname  *string          `json:"lastname"`
	Bio       *string          `json:"bio"`
	Location  *string          `json:"location"`
	Presence  PresenceResponse `json:"presence"`
}

// Both mappers take location separately rather than a database.Address,
// because a user may have no address row at all - the service passes an
// invalid NullString for that case instead of a zero-valued struct.
func ToOwnProfileResponse(u database.User, p database.Profile, location sql.NullString) OwnProfileResponse {
	return OwnProfileResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Firstname:   nullStringPtr(p.Firstname),
		Lastname:    nullStringPtr(p.Lastname),
		Bio:         nullStringPtr(p.Bio),
		PhoneNumber: nullStringPtr(p.PhoneNumber),
		DateOfBirth: nullDatePtr(p.DateOfBirth),
		Location:    nullStringPtr(location),
	}
}

func ToPublicProfileResponse(u database.User, p database.Profile, location sql.NullString) PublicProfileResponse {
	return PublicProfileResponse{
		ID:        u.ID,
		Username:  u.Username,
		Firstname: nullStringPtr(p.Firstname),
		Lastname:  nullStringPtr(p.Lastname),
		Bio:       nullStringPtr(p.Bio),
		Location:  nullStringPtr(location),
		Presence:  toPresence(u.LastSeenAt, u.ShowOnlineStatus),
	}
}

// nullStringPtr turns sqlc's NullString into the *string the JSON wants.
func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// nullDatePtr formats a DATE as YYYY-MM-DD, dropping the time entirely.
func nullDatePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(DateLayout)
	return &s
}
