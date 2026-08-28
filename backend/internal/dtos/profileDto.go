package dtos

import (
	"database/sql"

	"github.com/google/uuid"

	"github.com/IbnBaqqi/transcendence/internal/database"
)

const DateLayout = "2006-01-02"

type UpdateProfileInput struct {
	Firstname   OptionalString `json:"firstname"`
	Lastname    OptionalString `json:"lastname"`
	Bio         OptionalString `json:"bio"`
	PhoneNumber OptionalString `json:"phone_number"`
	DateOfBirth OptionalString `json:"date_of_birth"`
	Location    OptionalString `json:"location"`
}

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
	AvatarURL   *string   `json:"avatar_url"`
}

type AvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

type PublicProfileResponse struct {
	ID        uuid.UUID         `json:"id"`
	Username  string            `json:"username"`
	Firstname *string           `json:"firstname"`
	Lastname  *string           `json:"lastname"`
	Bio       *string           `json:"bio"`
	Location  *string           `json:"location"`
	AvatarURL *string           `json:"avatar_url"`
	Presence  *PresenceResponse `json:"presence,omitempty"`
}

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
		AvatarURL:   avatarURL(p.AvatarFilename),
	}
}

// includePresence is false for an unauthenticated caller, and then the field
// is omitted rather than sent as offline. Sending `{"is_online": false}` to
// everyone signed out would be a claim about the subject that is not true; an
// absent field says only "you are not signed in", which the caller knows.
//
// For a signed-in viewer presence stays present-but-offline when a block
// exists, and that ambiguity is deliberate - it is what stops the response
// announcing the block.
func ToPublicProfileResponse(u database.User, p database.Profile, location sql.NullString, includePresence bool) PublicProfileResponse {
	var presence *PresenceResponse
	if includePresence {
		p := toPresence(u.LastSeenAt, u.ShowOnlineStatus)
		presence = &p
	}

	return PublicProfileResponse{
		ID:        u.ID,
		Username:  u.Username,
		Firstname: nullStringPtr(p.Firstname),
		Lastname:  nullStringPtr(p.Lastname),
		Bio:       nullStringPtr(p.Bio),
		Location:  nullStringPtr(location),
		AvatarURL: avatarURL(p.AvatarFilename),
		Presence:  presence,
	}
}

func avatarURL(filename sql.NullString) *string {
	if !filename.Valid {
		return nil
	}
	url := UploadURLPrefix + filename.String
	return &url
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func nullDatePtr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(DateLayout)
	return &s
}
